package funcs

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestCryptoTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		in   any
		want string
	}{
		{"sha1", `{{ .Value | sha1 }}`, "abc", "a9993e364706816aba3e25717850c26c9cd0d89d"},
		{"sha1 empty", `{{ .Value | sha1 }}`, "", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"sha256", `{{ .Value | sha256 }}`, "abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"adler32", `{{ .Value | adler32 }}`, "abc", "38600999"},
		{"adler32 empty", `{{ .Value | adler32 }}`, "", "1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, tc.src, V(tc.in))
			if got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
}

func TestRandBytesTemplate(t *testing.T) {
	got := mustRender(t, `{{ randbytes .Value }}`, V(16))
	b, err := base64.StdEncoding.DecodeString(got)
	if err != nil || len(b) != 16 {
		t.Fatalf("decoded len = %d err = %v", len(b), err)
	}
	if got := mustRender(t, `{{ randbytes .Value }}`, V(0)); got != "" {
		t.Fatalf("zero want empty, got %q", got)
	}
	if got := mustRender(t, `{{ randbytes .Value }}`, V(-1)); got != "" {
		t.Fatalf("negative want empty, got %q", got)
	}
}

func TestEncryptDecryptAESTemplate_RoundTrip(t *testing.T) {
	plaintexts := []string{"", "hello", "hello world", strings.Repeat("data", 100)}
	for _, pt := range plaintexts {
		ct := mustRender(t, `{{ aes_encrypt "secret" .Value }}`, V(pt))
		got := mustRender(t, `{{ aes_decrypt "secret" .Value }}`, V(ct))
		if got != pt {
			t.Fatalf("roundtrip = %q, want %q", got, pt)
		}
	}
}

func TestDecryptAESTemplate_BadInput(t *testing.T) {
	if _, err := render(t, `{{ aes_decrypt "k" .Value }}`, V("@@not-base64@@")); err == nil {
		t.Fatalf("expected base64 error")
	}
	if _, err := render(t, `{{ aes_decrypt "k" .Value }}`, V("AAAA")); err == nil {
		t.Fatalf("expected short ciphertext error")
	}
	// 32 bytes of zeros: block-aligned but invalid PKCS7 padding after decrypt.
	junk := base64.StdEncoding.EncodeToString(make([]byte, 32))
	if _, err := render(t, `{{ aes_decrypt "k" .Value }}`, V(junk)); err == nil {
		t.Fatalf("expected padding error")
	}
	// 17 bytes: >= block size but not block-aligned.
	notAligned := base64.StdEncoding.EncodeToString(make([]byte, 17))
	if _, err := render(t, `{{ aes_decrypt "k" .Value }}`, V(notAligned)); err == nil {
		t.Fatalf("expected not-block-aligned error")
	}
}
