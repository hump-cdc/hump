package postgres

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// whereMaxDepth caps expression nesting so a crafted filter cannot exhaust the stack.
const whereMaxDepth = 64

// whereFunctions are the functions a filter may call: type conversions and
// cheap date, string and null helpers, none of which reads or writes state.
var whereFunctions = map[string]bool{
	"abs": true, "coalesce": true, "date_part": true, "date_trunc": true,
	"length": true, "lower": true, "now": true, "nullif": true, "to_char": true,
	"to_date": true, "to_number": true, "to_timestamp": true, "upper": true,
}

// whereTypes are the type names a filter may cast to. Multi-word types are
// keyed by their first word and completed by parseType.
var whereTypes = map[string]bool{
	"bigint": true, "bool": true, "boolean": true, "char": true, "character": true,
	"date": true, "decimal": true, "double": true, "float": true, "float4": true,
	"float8": true, "int": true, "int2": true, "int4": true, "int8": true,
	"integer": true, "interval": true, "json": true, "jsonb": true, "numeric": true,
	"real": true, "smallint": true, "text": true, "time": true, "timestamp": true,
	"timestamptz": true, "timetz": true, "uuid": true, "varchar": true,
}

// whereTypedLiterals are the types that may prefix a string literal, as in
// interval '1 day'.
var whereTypedLiterals = map[string]bool{
	"date": true, "interval": true, "time": true, "timestamp": true, "timestamptz": true,
}

// whereExtractFields are the fields extract accepts.
var whereExtractFields = map[string]bool{
	"century": true, "day": true, "decade": true, "dow": true, "doy": true,
	"epoch": true, "hour": true, "isodow": true, "isoyear": true, "microseconds": true,
	"millennium": true, "milliseconds": true, "minute": true, "month": true,
	"quarter": true, "second": true, "week": true, "year": true,
}

// whereComparisons are the binary comparison operators a filter may use.
var whereComparisons = map[string]bool{
	"=": true, "<>": true, "!=": true, "<": true, ">": true, "<=": true, ">=": true,
}

// whereTokenKind classifies a lexed token of a filter.
type whereTokenKind int

const (
	// whereTokenEnd marks the end of the input.
	whereTokenEnd whereTokenKind = iota
	// whereTokenWord is a bare identifier or keyword, lowercased.
	whereTokenWord
	// whereTokenQuoted is a double-quoted identifier, unescaped.
	whereTokenQuoted
	// whereTokenNumber is a numeric literal as written.
	whereTokenNumber
	// whereTokenString is a single-quoted string literal, unescaped.
	whereTokenString
	// whereTokenSymbol is an operator or punctuation.
	whereTokenSymbol
)

// whereToken is one lexed token and the byte offset it starts at.
type whereToken struct {
	kind whereTokenKind
	text string
	pos  int
}

// whereParser is a recursive descent parser over the tokens of one filter.
type whereParser struct {
	tokens []whereToken
	next   int
	depth  int
}

// parseWhere checks that where is a plain boolean expression of columns,
// literals, whitelisted operators, casts and functions, and renders it back as
// SQL with every identifier and literal quoted. An empty where renders as "".
func parseWhere(where string) (string, error) {
	tokens, err := lexWhere(where)
	if err != nil {
		return "", err
	}

	if len(tokens) == 1 {
		return "", nil
	}

	p := &whereParser{tokens: tokens}

	sql, err := p.parseOr()
	if err != nil {
		return "", err
	}

	if t := p.peek(); t.kind != whereTokenEnd {
		return "", p.unexpected(t)
	}

	return sql, nil
}

// lexWhere splits where into tokens ending with a whereTokenEnd, rejecting
// comments and every character outside the accepted syntax.
func lexWhere(where string) ([]whereToken, error) {
	var tokens []whereToken

	for i := 0; i < len(where); {
		c := where[i]

		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case isWordStart(c):
			start := i
			for i < len(where) && (isWordStart(where[i]) || isDigit(where[i])) {
				i++
			}
			tokens = append(tokens, whereToken{kind: whereTokenWord, text: strings.ToLower(where[start:i]), pos: start})
		case isDigit(c) || c == '.' && i+1 < len(where) && isDigit(where[i+1]):
			end, err := lexNumber(where, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, whereToken{kind: whereTokenNumber, text: where[i:end], pos: i})
			i = end
		case c == '\'' || c == '"':
			text, end, err := lexQuoted(where, i)
			if err != nil {
				return nil, err
			}

			kind := whereTokenString
			if c == '"' {
				kind = whereTokenQuoted
			}
			tokens = append(tokens, whereToken{kind: kind, text: text, pos: i})
			i = end
		case strings.HasPrefix(where[i:], "--") || strings.HasPrefix(where[i:], "/*"):
			return nil, fmt.Errorf("comment at offset %d is not allowed", i)
		default:
			symbol := lexSymbol(where[i:])
			if symbol == "" {
				return nil, fmt.Errorf("unexpected character %q at offset %d", c, i)
			}
			tokens = append(tokens, whereToken{kind: whereTokenSymbol, text: symbol, pos: i})
			i += len(symbol)
		}
	}

	return append(tokens, whereToken{kind: whereTokenEnd, pos: len(where)}), nil
}

// lexNumber returns the end of the numeric literal starting at start: digits
// with an optional fraction and exponent.
func lexNumber(where string, start int) (int, error) {
	i := start
	for i < len(where) && isDigit(where[i]) {
		i++
	}

	if i < len(where) && where[i] == '.' {
		i++
		for i < len(where) && isDigit(where[i]) {
			i++
		}
	}

	if i < len(where) && (where[i] == 'e' || where[i] == 'E') {
		i++
		if i < len(where) && (where[i] == '+' || where[i] == '-') {
			i++
		}

		digits := i
		for i < len(where) && isDigit(where[i]) {
			i++
		}

		if i == digits {
			return 0, fmt.Errorf("malformed number at offset %d", start)
		}
	}

	return i, nil
}

// lexQuoted unescapes the quoted token starting at start, whose quote doubles
// to escape itself, and returns its text and the offset past the closing quote.
func lexQuoted(where string, start int) (string, int, error) {
	quote := where[start]

	var text strings.Builder
	for i := start + 1; i < len(where); i++ {
		if where[i] != quote {
			text.WriteByte(where[i])
			continue
		}

		if i+1 < len(where) && where[i+1] == quote {
			text.WriteByte(quote)
			i++
			continue
		}

		if quote == '"' && text.Len() == 0 {
			return "", 0, fmt.Errorf("empty quoted identifier at offset %d", start)
		}

		return text.String(), i + 1, nil
	}

	return "", 0, fmt.Errorf("unterminated quote at offset %d", start)
}

// lexSymbol returns the operator or punctuation that prefixes s, or "" when s
// starts with none of them. Two-character symbols win over their prefixes.
func lexSymbol(s string) string {
	for _, symbol := range []string{"<=", ">=", "<>", "!=", "::", "||"} {
		if strings.HasPrefix(s, symbol) {
			return symbol
		}
	}

	if strings.IndexByte("=<>+-*/%(),.", s[0]) >= 0 {
		return s[:1]
	}

	return ""
}

// isWordStart reports whether c may start a bare identifier.
func isWordStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// isDigit reports whether c is an ASCII digit.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// parseOr parses a disjunction, the loosest-binding level of the grammar.
func (p *whereParser) parseOr() (string, error) {
	p.depth++
	defer func() { p.depth-- }()

	if p.depth > whereMaxDepth {
		return "", errors.New("expression is nested too deeply")
	}

	return p.parseChain("or", p.parseAnd)
}

// parseAnd parses a conjunction.
func (p *whereParser) parseAnd() (string, error) {
	return p.parseChain("and", p.parseNot)
}

// parseChain parses operands joined by the keyword, left-associatively.
func (p *whereParser) parseChain(keyword string, operand func() (string, error)) (string, error) {
	left, err := operand()
	if err != nil {
		return "", err
	}

	for p.acceptWord(keyword) {
		right, err := operand()
		if err != nil {
			return "", err
		}

		left = "(" + left + " " + strings.ToUpper(keyword) + " " + right + ")"
	}

	return left, nil
}

// parseNot parses a NOT prefix, which binds looser than any comparison.
func (p *whereParser) parseNot() (string, error) {
	if !p.acceptWord("not") {
		return p.parseComparison()
	}

	operand, err := p.parseNested(p.parseNot)
	if err != nil {
		return "", err
	}

	return "(NOT " + operand + ")", nil
}

// parseComparison parses one optional comparison, IS, IN, BETWEEN or LIKE test
// applied to a value. Comparisons do not chain.
func (p *whereParser) parseComparison() (string, error) {
	left, err := p.parseConcat()
	if err != nil {
		return "", err
	}

	t := p.peek()
	if t.kind == whereTokenSymbol && whereComparisons[t.text] {
		p.next++

		right, err := p.parseConcat()
		if err != nil {
			return "", err
		}

		return "(" + left + " " + t.text + " " + right + ")", nil
	}

	if p.acceptWord("is") {
		return p.parseIs(left)
	}

	negation := ""
	if p.isWord(p.peek(), "not") && p.isNegatable(p.peekAt(1)) {
		p.next++
		negation = "NOT "
	}

	switch {
	case p.acceptWord("in"):
		return p.parseIn(left, negation)
	case p.acceptWord("between"):
		return p.parseBetween(left, negation)
	case p.acceptWord("like"):
		return p.parsePattern(left, negation+"LIKE")
	case p.acceptWord("ilike"):
		return p.parsePattern(left, negation+"ILIKE")
	}

	return left, nil
}

// parseIs parses the rest of an IS test on left.
func (p *whereParser) parseIs(left string) (string, error) {
	negation := ""
	if p.acceptWord("not") {
		negation = "NOT "
	}

	for _, keyword := range []string{"null", "true", "false"} {
		if p.acceptWord(keyword) {
			return "(" + left + " IS " + negation + strings.ToUpper(keyword) + ")", nil
		}
	}

	if err := p.expectWord("distinct"); err != nil {
		return "", err
	}

	if err := p.expectWord("from"); err != nil {
		return "", err
	}

	right, err := p.parseConcat()
	if err != nil {
		return "", err
	}

	return "(" + left + " IS " + negation + "DISTINCT FROM " + right + ")", nil
}

// parseIn parses the parenthesized value list of an IN test on left.
func (p *whereParser) parseIn(left, negation string) (string, error) {
	if err := p.expectSymbol("("); err != nil {
		return "", err
	}

	values, err := p.parseList()
	if err != nil {
		return "", err
	}

	return "(" + left + " " + negation + "IN (" + values + "))", nil
}

// parseBetween parses the bounds of a BETWEEN test on left.
func (p *whereParser) parseBetween(left, negation string) (string, error) {
	low, err := p.parseConcat()
	if err != nil {
		return "", err
	}

	if err := p.expectWord("and"); err != nil {
		return "", err
	}

	high, err := p.parseConcat()
	if err != nil {
		return "", err
	}

	return "(" + left + " " + negation + "BETWEEN " + low + " AND " + high + ")", nil
}

// parsePattern parses the pattern of a LIKE or ILIKE test on left.
func (p *whereParser) parsePattern(left, operator string) (string, error) {
	pattern, err := p.parseConcat()
	if err != nil {
		return "", err
	}

	return "(" + left + " " + operator + " " + pattern + ")", nil
}

// parseConcat parses string concatenation, which binds looser than arithmetic.
func (p *whereParser) parseConcat() (string, error) {
	return p.parseBinary([]string{"||"}, p.parseAdditive)
}

// parseAdditive parses addition and subtraction.
func (p *whereParser) parseAdditive() (string, error) {
	return p.parseBinary([]string{"+", "-"}, p.parseMultiplicative)
}

// parseMultiplicative parses multiplication, division and modulo.
func (p *whereParser) parseMultiplicative() (string, error) {
	return p.parseBinary([]string{"*", "/", "%"}, p.parseUnary)
}

// parseBinary parses operands joined by any of operators, left-associatively.
func (p *whereParser) parseBinary(operators []string, operand func() (string, error)) (string, error) {
	left, err := operand()
	if err != nil {
		return "", err
	}

	for {
		t := p.peek()
		if t.kind != whereTokenSymbol || !slices.Contains(operators, t.text) {
			return left, nil
		}
		p.next++

		right, err := operand()
		if err != nil {
			return "", err
		}

		left = "(" + left + " " + t.text + " " + right + ")"
	}
}

// parseUnary parses a sign prefix. The operand is parenthesized so that two
// minus signs never render as a comment.
func (p *whereParser) parseUnary() (string, error) {
	t := p.peek()
	if t.kind != whereTokenSymbol || t.text != "-" && t.text != "+" {
		return p.parseCast()
	}
	p.next++

	operand, err := p.parseNested(p.parseUnary)
	if err != nil {
		return "", err
	}

	return "(" + t.text + "(" + operand + "))", nil
}

// parseCast parses a value followed by any number of ::type casts.
func (p *whereParser) parseCast() (string, error) {
	value, err := p.parsePrimary()
	if err != nil {
		return "", err
	}

	for p.acceptSymbol("::") {
		typ, err := p.parseType()
		if err != nil {
			return "", err
		}

		value = "(" + value + "::" + typ + ")"
	}

	return value, nil
}

// parsePrimary parses a literal, a column, a parenthesized expression, or one
// of the whitelisted special forms and function calls.
func (p *whereParser) parsePrimary() (string, error) {
	t := p.peek()

	switch t.kind {
	case whereTokenNumber:
		p.next++
		return t.text, nil
	case whereTokenString:
		p.next++
		return quoteLiteral(t.text), nil
	case whereTokenQuoted:
		return p.parseColumn()
	case whereTokenSymbol:
		if !p.acceptSymbol("(") {
			return "", p.unexpected(t)
		}

		inner, err := p.parseOr()
		if err != nil {
			return "", err
		}

		if err := p.expectSymbol(")"); err != nil {
			return "", err
		}

		return "(" + inner + ")", nil
	case whereTokenWord:
		return p.parseWord()
	}

	return "", p.unexpected(t)
}

// parseWord parses a primary that starts with a bare word: a keyword literal,
// a special form, a typed literal, a function call or a column.
func (p *whereParser) parseWord() (string, error) {
	t := p.peek()
	following := p.peekAt(1)

	switch {
	case t.text == "null" || t.text == "true" || t.text == "false" ||
		t.text == "current_date" || t.text == "current_timestamp":
		p.next++
		return strings.ToUpper(t.text), nil
	case t.text == "cast" && p.isSymbol(following, "("):
		return p.parseCastCall()
	case t.text == "extract" && p.isSymbol(following, "("):
		return p.parseExtract()
	case whereTypedLiterals[t.text] && following.kind == whereTokenString:
		p.next += 2
		return strings.ToUpper(t.text) + " " + quoteLiteral(following.text), nil
	case p.isSymbol(following, "("):
		return p.parseFunction()
	}

	return p.parseColumn()
}

// parseCastCall parses CAST(value AS type).
func (p *whereParser) parseCastCall() (string, error) {
	p.next += 2

	value, err := p.parseOr()
	if err != nil {
		return "", err
	}

	if err := p.expectWord("as"); err != nil {
		return "", err
	}

	typ, err := p.parseType()
	if err != nil {
		return "", err
	}

	if err := p.expectSymbol(")"); err != nil {
		return "", err
	}

	return "CAST(" + value + " AS " + typ + ")", nil
}

// parseExtract parses EXTRACT(field FROM value).
func (p *whereParser) parseExtract() (string, error) {
	p.next += 2

	field := p.peek()
	if field.kind != whereTokenWord || !whereExtractFields[field.text] {
		return "", p.unexpected(field)
	}
	p.next++

	if err := p.expectWord("from"); err != nil {
		return "", err
	}

	value, err := p.parseOr()
	if err != nil {
		return "", err
	}

	if err := p.expectSymbol(")"); err != nil {
		return "", err
	}

	return "EXTRACT(" + strings.ToUpper(field.text) + " FROM " + value + ")", nil
}

// parseFunction parses a call to a whitelisted function.
func (p *whereParser) parseFunction() (string, error) {
	name := p.peek()
	if !whereFunctions[name.text] {
		return "", fmt.Errorf("function %q at offset %d is not allowed", name.text, name.pos)
	}
	p.next += 2

	if p.acceptSymbol(")") {
		return name.text + "()", nil
	}

	args, err := p.parseList()
	if err != nil {
		return "", err
	}

	return name.text + "(" + args + ")", nil
}

// parseList parses a non-empty comma-separated list of expressions and its
// closing parenthesis.
func (p *whereParser) parseList() (string, error) {
	var items []string

	for {
		item, err := p.parseOr()
		if err != nil {
			return "", err
		}
		items = append(items, item)

		if !p.acceptSymbol(",") {
			break
		}
	}

	if err := p.expectSymbol(")"); err != nil {
		return "", err
	}

	return strings.Join(items, ", "), nil
}

// parseColumn parses a column reference, optionally qualified by its table.
func (p *whereParser) parseColumn() (string, error) {
	column := quoteIdentifier(p.peek().text)
	p.next++

	if !p.acceptSymbol(".") {
		return column, nil
	}

	t := p.peek()
	if t.kind != whereTokenWord && t.kind != whereTokenQuoted {
		return "", p.unexpected(t)
	}
	p.next++

	return column + "." + quoteIdentifier(t.text), nil
}

// parseType parses a whitelisted type name, including the multi-word ones, and
// an optional precision.
func (p *whereParser) parseType() (string, error) {
	t := p.peek()
	if t.kind != whereTokenWord || !whereTypes[t.text] {
		return "", fmt.Errorf("type at offset %d is not allowed", t.pos)
	}
	p.next++

	typ := t.text
	switch typ {
	case "double":
		if err := p.expectWord("precision"); err != nil {
			return "", err
		}
		typ += " precision"
	case "character":
		if p.acceptWord("varying") {
			typ += " varying"
		}
	}

	precision, err := p.parsePrecision()
	if err != nil {
		return "", err
	}
	typ += precision

	if t.text == "time" || t.text == "timestamp" {
		zone, err := p.parseTimeZone()
		if err != nil {
			return "", err
		}
		typ += zone
	}

	return typ, nil
}

// parsePrecision parses an optional parenthesized list of numeric type modifiers.
func (p *whereParser) parsePrecision() (string, error) {
	if !p.acceptSymbol("(") {
		return "", nil
	}

	var modifiers []string
	for {
		t := p.peek()
		if t.kind != whereTokenNumber {
			return "", p.unexpected(t)
		}
		p.next++
		modifiers = append(modifiers, t.text)

		if !p.acceptSymbol(",") {
			break
		}
	}

	if err := p.expectSymbol(")"); err != nil {
		return "", err
	}

	return "(" + strings.Join(modifiers, ",") + ")", nil
}

// parseTimeZone parses an optional WITH or WITHOUT TIME ZONE suffix.
func (p *whereParser) parseTimeZone() (string, error) {
	suffix := ""
	switch {
	case p.acceptWord("with"):
		suffix = " with time zone"
	case p.acceptWord("without"):
		suffix = " without time zone"
	default:
		return "", nil
	}

	if err := p.expectWord("time"); err != nil {
		return "", err
	}

	if err := p.expectWord("zone"); err != nil {
		return "", err
	}

	return suffix, nil
}

// parseNested runs parse one nesting level deeper, failing once the expression
// is nested more than whereMaxDepth levels.
func (p *whereParser) parseNested(parse func() (string, error)) (string, error) {
	p.depth++
	defer func() { p.depth-- }()

	if p.depth > whereMaxDepth {
		return "", errors.New("expression is nested too deeply")
	}

	return parse()
}

// peek returns the current token without consuming it.
func (p *whereParser) peek() whereToken {
	return p.peekAt(0)
}

// peekAt returns the token offset positions ahead, or the end token past it.
func (p *whereParser) peekAt(offset int) whereToken {
	return p.tokens[min(p.next+offset, len(p.tokens)-1)]
}

// isWord reports whether t is the bare keyword.
func (p *whereParser) isWord(t whereToken, keyword string) bool {
	return t.kind == whereTokenWord && t.text == keyword
}

// isSymbol reports whether t is the symbol.
func (p *whereParser) isSymbol(t whereToken, symbol string) bool {
	return t.kind == whereTokenSymbol && t.text == symbol
}

// isNegatable reports whether t is a test NOT may prefix after a value.
func (p *whereParser) isNegatable(t whereToken) bool {
	return t.kind == whereTokenWord && (t.text == "in" || t.text == "between" || t.text == "like" || t.text == "ilike")
}

// acceptWord consumes the current token when it is the bare keyword.
func (p *whereParser) acceptWord(keyword string) bool {
	if !p.isWord(p.peek(), keyword) {
		return false
	}
	p.next++

	return true
}

// acceptSymbol consumes the current token when it is the symbol.
func (p *whereParser) acceptSymbol(symbol string) bool {
	if !p.isSymbol(p.peek(), symbol) {
		return false
	}
	p.next++

	return true
}

// expectWord consumes the bare keyword or fails.
func (p *whereParser) expectWord(keyword string) error {
	if !p.acceptWord(keyword) {
		return p.unexpected(p.peek())
	}

	return nil
}

// expectSymbol consumes the symbol or fails.
func (p *whereParser) expectSymbol(symbol string) error {
	if !p.acceptSymbol(symbol) {
		return p.unexpected(p.peek())
	}

	return nil
}

// unexpected reports t as a token the grammar does not allow where it stands.
func (p *whereParser) unexpected(t whereToken) error {
	if t.kind == whereTokenEnd {
		return fmt.Errorf("unexpected end of expression at offset %d", t.pos)
	}

	return fmt.Errorf("unexpected %q at offset %d", t.text, t.pos)
}
