package funcs

import (
	"fmt"
	"net/url"
)

// tmplURLParse splits raw into a map with scheme, host, path, query,
// opaque, fragment, and userinfo entries. Mirrors sprig urlparse.
func tmplURLParse(raw string) (map[string]any, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	m := map[string]any{
		"scheme":   u.Scheme,
		"host":     u.Host,
		"hostname": u.Hostname(),
		"path":     u.Path,
		"query":    u.RawQuery,
		"opaque":   u.Opaque,
		"fragment": u.Fragment,
		"userinfo": "",
	}
	if u.User != nil {
		m["userinfo"] = u.User.String()
	}
	return m, nil
}

// tmplURLJoin reconstructs a URL string from a map produced by tmplURLParse
// (only the standard keys are honoured).
func tmplURLJoin(m map[string]any) (string, error) {
	get := func(k string) string {
		v, ok := m[k]
		if !ok || v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
	u := &url.URL{
		Scheme:   get("scheme"),
		Host:     get("host"),
		Path:     get("path"),
		RawQuery: get("query"),
		Opaque:   get("opaque"),
		Fragment: get("fragment"),
	}
	if userinfo := get("userinfo"); userinfo != "" {
		parsed, err := url.Parse("//" + userinfo + "@host")
		if err != nil {
			return "", err
		}
		u.User = parsed.User
	}
	return u.String(), nil
}
