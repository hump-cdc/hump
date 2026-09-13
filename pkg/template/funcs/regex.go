package funcs

import "regexp"

// tmplRegexMatch reports whether s matches the regular expression pattern.
func tmplRegexMatch(pattern, s string) (bool, error) {
	return regexp.MatchString(pattern, s)
}

// tmplRegexFind returns the leftmost match of pattern in s, or "" when no
// match is found.
func tmplRegexFind(pattern, s string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	return re.FindString(s), nil
}

// tmplRegexFindAll returns up to n matches of pattern in s. n=-1 returns all.
func tmplRegexFindAll(pattern string, n int, s string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.FindAllString(s, n), nil
}

// tmplRegexReplaceAll replaces every match of pattern in s with repl. The
// replacement string supports $-expansion (re.ReplaceAllString).
func tmplRegexReplaceAll(pattern, repl, s string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	return re.ReplaceAllString(s, repl), nil
}

// tmplRegexReplaceAllLiteral replaces matches with repl literally (no
// $-expansion).
func tmplRegexReplaceAllLiteral(pattern, repl, s string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	return re.ReplaceAllLiteralString(s, repl), nil
}

// tmplRegexSplit splits s on pattern up to n times. n=-1 returns all parts.
func tmplRegexSplit(pattern string, n int, s string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.Split(s, n), nil
}
