package funcs

import (
	"strings"
	"unicode"
)

// splitWords tokenizes s into words by splitting on non-alphanumeric runes
// and on case/digit boundaries: lower→Upper ("fooBar"), digit↔letter
// ("foo2bar"), and the trailing acronym→Title boundary ("XMLParser" →
// "XML","Parser"). Empty input yields nil.
func splitWords(s string) []string {
	if s == "" {
		return nil
	}
	runes := []rune(s)
	var words []string
	start := 0
	flush := func(end int) {
		if end > start {
			words = append(words, string(runes[start:end]))
		}
		start = end
	}
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush(i)
			start = i + 1
			continue
		}
		if i == 0 {
			continue
		}
		prev := runes[i-1]
		switch {
		case unicode.IsLower(prev) && unicode.IsUpper(r):
			flush(i)
		case unicode.IsDigit(prev) != unicode.IsDigit(r):
			flush(i)
		case unicode.IsUpper(prev) && unicode.IsUpper(r) && i+1 < len(runes) && unicode.IsLower(runes[i+1]):
			flush(i)
		}
	}
	flush(len(runes))
	return words
}

// titleWord returns s with its first rune uppercased and the rest lowercased.
// Used to normalize a tokenized word to Title form regardless of its original
// casing (so acronyms like "URL" become "Url").
func titleWord(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	for i := 1; i < len(r); i++ {
		r[i] = unicode.ToLower(r[i])
	}
	return string(r)
}

// joinLower lowercases every word and joins them with sep.
func joinLower(words []string, sep string) string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = strings.ToLower(w)
	}
	return strings.Join(out, sep)
}

// joinUpper uppercases every word and joins them with sep.
func joinUpper(words []string, sep string) string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = strings.ToUpper(w)
	}
	return strings.Join(out, sep)
}

// joinTitle title-cases every word (see titleWord) and joins them with sep.
func joinTitle(words []string, sep string) string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = titleWord(w)
	}
	return strings.Join(out, sep)
}

// tmplCamelCase renders s as camelCase: first word fully lowercase, each
// subsequent word title-cased, no separator. "getURLPath" → "getUrlPath".
// A nil source yields "".
func tmplCamelCase(s any) string {
	words := splitWords(strArg(s))
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(words[0]))
	for _, w := range words[1:] {
		b.WriteString(titleWord(w))
	}
	return b.String()
}

// tmplPascalCase renders s as PascalCase: every word title-cased, no
// separator. "get url path" → "GetUrlPath". A nil source yields "".
func tmplPascalCase(s any) string {
	var b strings.Builder
	for _, w := range splitWords(strArg(s)) {
		b.WriteString(titleWord(w))
	}
	return b.String()
}

// tmplSnakeCase renders s as snake_case: lowercase words joined by "_".
// A nil source yields "".
func tmplSnakeCase(s any) string { return joinLower(splitWords(strArg(s)), "_") }

// tmplConstantCase renders s as CONSTANT_CASE: uppercase words joined by "_".
// A nil source yields "".
func tmplConstantCase(s any) string { return joinUpper(splitWords(strArg(s)), "_") }

// tmplKebabCase renders s as kebab-case: lowercase words joined by "-".
// A nil source yields "".
func tmplKebabCase(s any) string { return joinLower(splitWords(strArg(s)), "-") }

// tmplUpperCase renders s as space-separated UPPER words. A nil source
// yields "".
func tmplUpperCase(s any) string { return joinUpper(splitWords(strArg(s)), " ") }

// tmplLowerCase renders s as space-separated lower words. A nil source
// yields "".
func tmplLowerCase(s any) string { return joinLower(splitWords(strArg(s)), " ") }

// tmplCapitalCase renders s as space-separated Title-cased words. A nil
// source yields "".
func tmplCapitalCase(s any) string { return joinTitle(splitWords(strArg(s)), " ") }

// tmplDotCase renders s as dot.case: lowercase words joined by ".".
// A nil source yields "".
func tmplDotCase(s any) string { return joinLower(splitWords(strArg(s)), ".") }

// tmplHeaderCase renders s as Header-Case (a.k.a. Train-Case): Title-cased
// words joined by "-". A nil source yields "".
func tmplHeaderCase(s any) string { return joinTitle(splitWords(strArg(s)), "-") }

// tmplPathCase renders s as path/case: lowercase words joined by "/".
// A nil source yields "".
func tmplPathCase(s any) string { return joinLower(splitWords(strArg(s)), "/") }

// tmplPascalSnakeCase renders s as Pascal_Snake_Case: Title-cased words
// joined by "_". A nil source yields "".
func tmplPascalSnakeCase(s any) string { return joinTitle(splitWords(strArg(s)), "_") }

// tmplLowerFirst lowercases only the first rune of s; the rest is kept
// as-is. Unlike the multi-word casers it performs no tokenization. A nil
// source yields "".
func tmplLowerFirst(s any) string {
	str := strArg(s)
	if str == "" {
		return ""
	}
	r := []rune(str)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// tmplUpperFirst uppercases only the first rune of s; the rest is kept
// as-is. Unlike the multi-word casers it performs no tokenization. A nil
// source yields "".
func tmplUpperFirst(s any) string {
	str := strArg(s)
	if str == "" {
		return ""
	}
	r := []rune(str)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
