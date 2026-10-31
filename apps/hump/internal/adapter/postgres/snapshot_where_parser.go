package postgres

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// snapshotWhereMaxDepth caps expression nesting so a crafted filter cannot
// exhaust the stack.
const snapshotWhereMaxDepth = 64

// snapshotWhereFunctions are the functions a filter may call: type conversions and
// cheap date, string and null helpers, none of which reads or writes state.
var snapshotWhereFunctions = map[string]bool{
	"abs": true, "coalesce": true, "date_part": true, "date_trunc": true,
	"length": true, "lower": true, "now": true, "nullif": true, "to_char": true,
	"to_date": true, "to_number": true, "to_timestamp": true, "upper": true,
}

// snapshotWhereTypes are the type names a filter may cast to. Multi-word types are
// keyed by their first word and completed by parseType.
var snapshotWhereTypes = map[string]bool{
	"bigint": true, "bool": true, "boolean": true, "char": true, "character": true,
	"date": true, "decimal": true, "double": true, "float": true, "float4": true,
	"float8": true, "int": true, "int2": true, "int4": true, "int8": true,
	"integer": true, "interval": true, "json": true, "jsonb": true, "numeric": true,
	"real": true, "smallint": true, "text": true, "time": true, "timestamp": true,
	"timestamptz": true, "timetz": true, "uuid": true, "varchar": true,
}

// snapshotWhereTypedLiterals are the types that may prefix a string literal, as in
// interval '1 day'.
var snapshotWhereTypedLiterals = map[string]bool{
	"date": true, "interval": true, "time": true, "timestamp": true, "timestamptz": true,
}

// snapshotWhereExtractFields are the fields extract accepts.
var snapshotWhereExtractFields = map[string]bool{
	"century": true, "day": true, "decade": true, "dow": true, "doy": true,
	"epoch": true, "hour": true, "isodow": true, "isoyear": true, "microseconds": true,
	"millennium": true, "milliseconds": true, "minute": true, "month": true,
	"quarter": true, "second": true, "week": true, "year": true,
}

// snapshotWhereComparisons are the binary comparison operators a filter may use.
var snapshotWhereComparisons = map[string]bool{
	"=": true, "<>": true, "!=": true, "<": true, ">": true, "<=": true, ">=": true,
}

// snapshotWhereTokenKind classifies a lexed token of a filter.
type snapshotWhereTokenKind int

const (
	// snapshotWhereTokenEnd marks the end of the input.
	snapshotWhereTokenEnd snapshotWhereTokenKind = iota
	// snapshotWhereTokenWord is a bare identifier or keyword, lowercased.
	snapshotWhereTokenWord
	// snapshotWhereTokenQuoted is a double-quoted identifier, unescaped.
	snapshotWhereTokenQuoted
	// snapshotWhereTokenNumber is a numeric literal as written.
	snapshotWhereTokenNumber
	// snapshotWhereTokenString is a single-quoted string literal, unescaped.
	snapshotWhereTokenString
	// snapshotWhereTokenSymbol is an operator or punctuation.
	snapshotWhereTokenSymbol
)

// snapshotWhereToken is one lexed token and the byte offset it starts at.
type snapshotWhereToken struct {
	kind snapshotWhereTokenKind
	text string
	pos  int
}

// snapshotWhereLexer splits one filter into tokens.
type snapshotWhereLexer struct {
	input  string
	pos    int // pos is the byte offset of the next unread character.
	tokens []snapshotWhereToken
}

// snapshotWhereParser is a recursive descent parser over the tokens of one filter.
type snapshotWhereParser struct {
	tokens          []snapshotWhereToken
	next            int
	depth           int
	columnQualifier string // columnQualifier rebases column references onto the snapshot source alias when set.
	tableQualifiers map[string]bool
}

// newSnapshotWhereLexer returns a lexer at the start of where.
func newSnapshotWhereLexer(where string) *snapshotWhereLexer {
	return &snapshotWhereLexer{input: where}
}

// lex splits the input into tokens ending with a snapshotWhereTokenEnd, rejecting
// comments and every character outside the accepted syntax.
func (l *snapshotWhereLexer) lex() ([]snapshotWhereToken, error) {
	for l.pos < len(l.input) {
		c := l.input[l.pos]

		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			l.pos++
		case l.isWordStart(c):
			start := l.pos
			for l.pos < len(l.input) && (l.isWordStart(l.input[l.pos]) || l.isDigit(l.input[l.pos])) {
				l.pos++
			}
			l.tokens = append(l.tokens, snapshotWhereToken{kind: snapshotWhereTokenWord, text: strings.ToLower(l.input[start:l.pos]), pos: start})
		case l.isDigit(c) || c == '.' && l.pos+1 < len(l.input) && l.isDigit(l.input[l.pos+1]):
			if err := l.number(); err != nil {
				return nil, err
			}
		case c == '\'' || c == '"':
			if err := l.quoted(); err != nil {
				return nil, err
			}
		case strings.HasPrefix(l.input[l.pos:], "--") || strings.HasPrefix(l.input[l.pos:], "/*"):
			return nil, fmt.Errorf("comment at offset %d is not allowed", l.pos)
		default:
			symbol := l.symbol()
			if symbol == "" {
				return nil, fmt.Errorf("unexpected character %q at offset %d", c, l.pos)
			}
			l.tokens = append(l.tokens, snapshotWhereToken{kind: snapshotWhereTokenSymbol, text: symbol, pos: l.pos})
			l.pos += len(symbol)
		}
	}

	return append(l.tokens, snapshotWhereToken{kind: snapshotWhereTokenEnd, pos: len(l.input)}), nil
}

// number reads the numeric literal at pos: digits with an optional fraction
// and exponent.
func (l *snapshotWhereLexer) number() error {
	i := l.pos
	for i < len(l.input) && l.isDigit(l.input[i]) {
		i++
	}

	if i < len(l.input) && l.input[i] == '.' {
		i++
		for i < len(l.input) && l.isDigit(l.input[i]) {
			i++
		}
	}

	if i < len(l.input) && (l.input[i] == 'e' || l.input[i] == 'E') {
		i++
		if i < len(l.input) && (l.input[i] == '+' || l.input[i] == '-') {
			i++
		}

		digits := i
		for i < len(l.input) && l.isDigit(l.input[i]) {
			i++
		}

		if i == digits {
			return fmt.Errorf("malformed number at offset %d", l.pos)
		}
	}

	l.tokens = append(l.tokens, snapshotWhereToken{kind: snapshotWhereTokenNumber, text: l.input[l.pos:i], pos: l.pos})
	l.pos = i

	return nil
}

// quoted reads the quoted token at pos, a string literal or a quoted
// identifier, unescaping the quote that doubles to escape itself.
func (l *snapshotWhereLexer) quoted() error {
	quote := l.input[l.pos]

	kind := snapshotWhereTokenString
	if quote == '"' {
		kind = snapshotWhereTokenQuoted
	}

	var text strings.Builder
	for i := l.pos + 1; i < len(l.input); i++ {
		if l.input[i] != quote {
			text.WriteByte(l.input[i])
			continue
		}

		if i+1 < len(l.input) && l.input[i+1] == quote {
			text.WriteByte(quote)
			i++
			continue
		}

		if quote == '"' && text.Len() == 0 {
			return fmt.Errorf("empty quoted identifier at offset %d", l.pos)
		}

		l.tokens = append(l.tokens, snapshotWhereToken{kind: kind, text: text.String(), pos: l.pos})
		l.pos = i + 1

		return nil
	}

	return fmt.Errorf("unterminated quote at offset %d", l.pos)
}

// symbol returns the operator or punctuation at pos, or "" when there is none.
// Two-character symbols win over their prefixes.
func (l *snapshotWhereLexer) symbol() string {
	rest := l.input[l.pos:]

	for _, symbol := range []string{"<=", ">=", "<>", "!=", "::", "||"} {
		if strings.HasPrefix(rest, symbol) {
			return symbol
		}
	}

	if strings.IndexByte("=<>+-*/%(),.", rest[0]) >= 0 {
		return rest[:1]
	}

	return ""
}

// isWordStart reports whether c may start a bare identifier.
func (l *snapshotWhereLexer) isWordStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// isDigit reports whether c is an ASCII digit.
func (l *snapshotWhereLexer) isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// newSnapshotWhereParser returns a parser ready for Parse.
func newSnapshotWhereParser() *snapshotWhereParser {
	return &snapshotWhereParser{}
}

// Parse checks that where is a plain boolean expression of columns, literals,
// whitelisted operators, casts and functions, and renders it back as SQL with
// every identifier and literal quoted. An empty where renders as "".
func (p *snapshotWhereParser) Parse(where string) (string, error) {
	tokens, err := newSnapshotWhereLexer(where).lex()
	if err != nil {
		return "", err
	}

	if len(tokens) == 1 {
		return "", nil
	}

	p.tokens, p.next, p.depth = tokens, 0, 0

	sql, err := p.parseOr()
	if err != nil {
		return "", err
	}

	if t := p.peek(); t.kind != snapshotWhereTokenEnd {
		return "", p.unexpected(t)
	}

	return sql, nil
}

// parseOr parses a disjunction, the loosest-binding level of the grammar.
func (p *snapshotWhereParser) parseOr() (string, error) {
	p.depth++
	defer func() { p.depth-- }()

	if p.depth > snapshotWhereMaxDepth {
		return "", errors.New("expression is nested too deeply")
	}

	return p.parseChain("or", p.parseAnd)
}

// parseAnd parses a conjunction.
func (p *snapshotWhereParser) parseAnd() (string, error) {
	return p.parseChain("and", p.parseNot)
}

// parseChain parses operands joined by the keyword, left-associatively.
func (p *snapshotWhereParser) parseChain(keyword string, operand func() (string, error)) (string, error) {
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
func (p *snapshotWhereParser) parseNot() (string, error) {
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
func (p *snapshotWhereParser) parseComparison() (string, error) {
	left, err := p.parseConcat()
	if err != nil {
		return "", err
	}

	t := p.peek()
	if t.kind == snapshotWhereTokenSymbol && snapshotWhereComparisons[t.text] {
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
func (p *snapshotWhereParser) parseIs(left string) (string, error) {
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
func (p *snapshotWhereParser) parseIn(left, negation string) (string, error) {
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
func (p *snapshotWhereParser) parseBetween(left, negation string) (string, error) {
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
func (p *snapshotWhereParser) parsePattern(left, operator string) (string, error) {
	pattern, err := p.parseConcat()
	if err != nil {
		return "", err
	}

	return "(" + left + " " + operator + " " + pattern + ")", nil
}

// parseConcat parses string concatenation, which binds looser than arithmetic.
func (p *snapshotWhereParser) parseConcat() (string, error) {
	return p.parseBinary([]string{"||"}, p.parseAdditive)
}

// parseAdditive parses addition and subtraction.
func (p *snapshotWhereParser) parseAdditive() (string, error) {
	return p.parseBinary([]string{"+", "-"}, p.parseMultiplicative)
}

// parseMultiplicative parses multiplication, division and modulo.
func (p *snapshotWhereParser) parseMultiplicative() (string, error) {
	return p.parseBinary([]string{"*", "/", "%"}, p.parseUnary)
}

// parseBinary parses operands joined by any of operators, left-associatively.
func (p *snapshotWhereParser) parseBinary(operators []string, operand func() (string, error)) (string, error) {
	left, err := operand()
	if err != nil {
		return "", err
	}

	for {
		t := p.peek()
		if t.kind != snapshotWhereTokenSymbol || !slices.Contains(operators, t.text) {
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
func (p *snapshotWhereParser) parseUnary() (string, error) {
	t := p.peek()
	if t.kind != snapshotWhereTokenSymbol || t.text != "-" && t.text != "+" {
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
func (p *snapshotWhereParser) parseCast() (string, error) {
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
func (p *snapshotWhereParser) parsePrimary() (string, error) {
	t := p.peek()

	switch t.kind {
	case snapshotWhereTokenNumber:
		p.next++
		return t.text, nil
	case snapshotWhereTokenString:
		p.next++
		return quoteLiteral(t.text), nil
	case snapshotWhereTokenQuoted:
		return p.parseColumn()
	case snapshotWhereTokenSymbol:
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
	case snapshotWhereTokenWord:
		return p.parseWord()
	default:
		return "", p.unexpected(t)
	}
}

// parseWord parses a primary that starts with a bare word: a keyword literal,
// a special form, a typed literal, a function call or a column.
func (p *snapshotWhereParser) parseWord() (string, error) {
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
	case snapshotWhereTypedLiterals[t.text] && following.kind == snapshotWhereTokenString:
		p.next += 2
		return strings.ToUpper(t.text) + " " + quoteLiteral(following.text), nil
	case p.isSymbol(following, "("):
		return p.parseFunction()
	}

	return p.parseColumn()
}

// parseCastCall parses CAST(value AS type).
func (p *snapshotWhereParser) parseCastCall() (string, error) {
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
func (p *snapshotWhereParser) parseExtract() (string, error) {
	p.next += 2

	field := p.peek()
	if field.kind != snapshotWhereTokenWord || !snapshotWhereExtractFields[field.text] {
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
func (p *snapshotWhereParser) parseFunction() (string, error) {
	name := p.peek()
	if !snapshotWhereFunctions[name.text] {
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
func (p *snapshotWhereParser) parseList() (string, error) {
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
func (p *snapshotWhereParser) parseColumn() (string, error) {
	qualifier := p.peek().text
	column := quoteIdentifier(qualifier)
	p.next++

	if !p.acceptSymbol(".") {
		if p.columnQualifier != "" {
			return quoteIdentifier(p.columnQualifier) + "." + column, nil
		}
		return column, nil
	}

	t := p.peek()
	if t.kind != snapshotWhereTokenWord && t.kind != snapshotWhereTokenQuoted {
		return "", p.unexpected(t)
	}
	p.next++

	if p.columnQualifier != "" && p.tableQualifiers[qualifier] {
		column = quoteIdentifier(p.columnQualifier)
	}
	return column + "." + quoteIdentifier(t.text), nil
}

// parseType parses a whitelisted type name, including the multi-word ones, and
// an optional precision.
func (p *snapshotWhereParser) parseType() (string, error) {
	t := p.peek()
	if t.kind != snapshotWhereTokenWord || !snapshotWhereTypes[t.text] {
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
func (p *snapshotWhereParser) parsePrecision() (string, error) {
	if !p.acceptSymbol("(") {
		return "", nil
	}

	var modifiers []string
	for {
		t := p.peek()
		if t.kind != snapshotWhereTokenNumber {
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
func (p *snapshotWhereParser) parseTimeZone() (string, error) {
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
// is nested more than snapshotWhereMaxDepth levels.
func (p *snapshotWhereParser) parseNested(parse func() (string, error)) (string, error) {
	p.depth++
	defer func() { p.depth-- }()

	if p.depth > snapshotWhereMaxDepth {
		return "", errors.New("expression is nested too deeply")
	}

	return parse()
}

// peek returns the current token without consuming it.
func (p *snapshotWhereParser) peek() snapshotWhereToken {
	return p.peekAt(0)
}

// peekAt returns the token offset positions ahead, or the end token past it.
func (p *snapshotWhereParser) peekAt(offset int) snapshotWhereToken {
	return p.tokens[min(p.next+offset, len(p.tokens)-1)]
}

// isWord reports whether t is the bare keyword.
func (p *snapshotWhereParser) isWord(t snapshotWhereToken, keyword string) bool {
	return t.kind == snapshotWhereTokenWord && t.text == keyword
}

// isSymbol reports whether t is the symbol.
func (p *snapshotWhereParser) isSymbol(t snapshotWhereToken, symbol string) bool {
	return t.kind == snapshotWhereTokenSymbol && t.text == symbol
}

// isNegatable reports whether t is a test NOT may prefix after a value.
func (p *snapshotWhereParser) isNegatable(t snapshotWhereToken) bool {
	return t.kind == snapshotWhereTokenWord && (t.text == "in" || t.text == "between" || t.text == "like" || t.text == "ilike")
}

// acceptWord consumes the current token when it is the bare keyword.
func (p *snapshotWhereParser) acceptWord(keyword string) bool {
	if !p.isWord(p.peek(), keyword) {
		return false
	}
	p.next++

	return true
}

// acceptSymbol consumes the current token when it is the symbol.
func (p *snapshotWhereParser) acceptSymbol(symbol string) bool {
	if !p.isSymbol(p.peek(), symbol) {
		return false
	}
	p.next++

	return true
}

// expectWord consumes the bare keyword or fails.
func (p *snapshotWhereParser) expectWord(keyword string) error {
	if !p.acceptWord(keyword) {
		return p.unexpected(p.peek())
	}

	return nil
}

// expectSymbol consumes the symbol or fails.
func (p *snapshotWhereParser) expectSymbol(symbol string) error {
	if !p.acceptSymbol(symbol) {
		return p.unexpected(p.peek())
	}

	return nil
}

// unexpected reports t as a token the grammar does not allow where it stands.
func (p *snapshotWhereParser) unexpected(t snapshotWhereToken) error {
	if t.kind == snapshotWhereTokenEnd {
		return fmt.Errorf("unexpected end of expression at offset %d", t.pos)
	}

	return fmt.Errorf("unexpected %q at offset %d", t.text, t.pos)
}
