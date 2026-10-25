package funcs

import (
	"encoding/base64"
	"errors"
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

func FuzzAESRoundTrip(f *testing.F) {
	f.Add("secret", "hello world")
	f.Add("", "")
	f.Add("k", strings.Repeat("x", 16))
	f.Add("\x00\xff", "\x10\x10\x10\x10")

	f.Fuzz(func(t *testing.T, key, plaintext string) {
		data := map[string]any{"Key": key, "Value": plaintext}

		ciphertext := mustRender(t, `{{ aes_encrypt .Key .Value }}`, data)

		data["Value"] = ciphertext
		if got := mustRender(t, `{{ aes_decrypt .Key .Value }}`, data); got != plaintext {
			t.Fatalf("aes_decrypt(aes_encrypt(%q)) = %q", plaintext, got)
		}
	})
}

func FuzzDecryptAES(f *testing.F) {
	f.Add("k", "@@not-base64@@")
	f.Add("k", "AAAA")
	f.Add("k", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	f.Add("k", base64.StdEncoding.EncodeToString(make([]byte, 17)))

	f.Fuzz(func(t *testing.T, key, ciphertext string) {
		// Called directly: text/template turns a panic in a func into an
		// error, which would hide it among the expected decryption failures.
		got, err := tmplDecryptAES(key, ciphertext)
		if err != nil {
			return
		}

		raw, err := base64.StdEncoding.DecodeString(ciphertext)
		if err != nil {
			t.Fatalf("tmplDecryptAES(%q) = %q, want an error for invalid base64", ciphertext, got)
		}

		if len(got) > len(raw)-16 {
			t.Fatalf("tmplDecryptAES(%q) = %d bytes, more than the %d-byte ciphertext minus the IV", ciphertext, len(got), len(raw))
		}
	})
}

func TestPkcs7Unpad(t *testing.T) {
	block := func(last byte, fill byte) []byte {
		b := []byte(strings.Repeat(string(fill), 15))
		return append(b, last)
	}
	cases := []struct {
		name string
		in   []byte
	}{
		{"empty", nil},
		{"not block-aligned", []byte{1, 2, 3}},
		{"zero pad", block(0, 0)},
		{"pad over block size", block(17, 17)},
		{"inconsistent pad", block(2, 'x')},
	}
	for _, tc := range cases {
		if _, err := pkcs7Unpad(tc.in, 16); err == nil {
			t.Errorf("pkcs7Unpad(%s): expected error", tc.name)
		}
	}
}

func TestCryptoTemplatesPropagateRandError(t *testing.T) {
	withFailingRand(t)

	for _, src := range []string{`{{ randbytes 4 }}`, `{{ aes_encrypt "k" "p" }}`} {
		if _, err := render(t, src, nil); !errors.Is(err, errRandFailed) {
			t.Errorf("%s error = %v, want errRandFailed", src, err)
		}
	}
}
