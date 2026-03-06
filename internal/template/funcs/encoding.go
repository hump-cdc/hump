package funcs

import (
	"encoding/base32"
	"encoding/base64"
)

// tmplB64Enc returns the standard base64 encoding of s.
func tmplB64Enc(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// tmplB64Dec decodes s as standard base64 and returns the resulting string.
// Decoding errors yield an empty string (matches sprig's quiet decode).
func tmplB64Dec(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(b)
}

// tmplB32Enc returns the standard base32 encoding of s.
func tmplB32Enc(s string) string {
	return base32.StdEncoding.EncodeToString([]byte(s))
}

// tmplB32Dec decodes s as standard base32 and returns the resulting string.
func tmplB32Dec(s string) string {
	b, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(b)
}
