package funcs

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash/adler32"
	"io"
	"strconv"
)

// tmplSHA256Sum returns the hex SHA-256 digest of s.
func tmplSHA256Sum(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// tmplAdler32Sum returns the Adler-32 checksum of s as a decimal string.
func tmplAdler32Sum(s string) string {
	return strconv.FormatUint(uint64(adler32.Checksum([]byte(s))), 10)
}

// tmplRandBytes returns n cryptographically-random bytes encoded as standard
// base64. Errors propagate to the template engine.
func tmplRandBytes(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// tmplEncryptAES encrypts plaintext with AES-256-CBC and returns the
// base64-encoded ciphertext. The IV is prepended to the ciphertext. key
// must be exactly 32 bytes after SHA-256 derivation: any string is hashed
// with SHA-256 to produce the encryption key (matches sprig).
func tmplEncryptAES(key, plaintext string) (string, error) {
	k := sha256.Sum256([]byte(key))
	// k is always 32 bytes (AES-256), so NewCipher never returns an error.
	block, _ := aes.NewCipher(k[:])

	pt := pkcs7Pad([]byte(plaintext))
	ct := make([]byte, aes.BlockSize+len(pt))
	iv := ct[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(block, iv) //nolint:gosec // iv is filled from crypto/rand above
	mode.CryptBlocks(ct[aes.BlockSize:], pt)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// tmplDecryptAES reverses tmplEncryptAES. The input is base64-decoded, the
// first block is the IV, and the remainder is decrypted with AES-256-CBC.
func tmplDecryptAES(key, b64 string) (string, error) {
	k := sha256.Sum256([]byte(key))
	// k is always 32 bytes (AES-256), so NewCipher never returns an error.
	block, _ := aes.NewCipher(k[:])

	ct, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	if len(ct) < aes.BlockSize || len(ct)%aes.BlockSize != 0 {
		return "", errors.New("ciphertext too short or not block-aligned")
	}
	iv := ct[:aes.BlockSize]
	ct = ct[aes.BlockSize:]
	pt := make([]byte, len(ct))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(pt, ct)
	pt, err = pkcs7Unpad(pt, aes.BlockSize)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// pkcs7Pad appends PKCS#7 padding to b up to the next multiple of aes.BlockSize.
func pkcs7Pad(b []byte) []byte {
	pad := byte(aes.BlockSize - len(b)%aes.BlockSize)
	return append(bytes.Clone(b), bytes.Repeat([]byte{pad}, int(pad))...)
}

// pkcs7Unpad strips PKCS#7 padding from b, rejecting a malformed pad.
func pkcs7Unpad(b []byte, blockSize int) ([]byte, error) {
	n := len(b)
	if n == 0 || n%blockSize != 0 {
		return nil, errors.New("invalid pkcs7 length")
	}
	pad := int(b[n-1])
	if pad == 0 || pad > blockSize {
		return nil, errors.New("invalid pkcs7 padding")
	}
	for i := n - pad; i < n; i++ {
		if int(b[i]) != pad {
			return nil, errors.New("invalid pkcs7 padding")
		}
	}
	return b[:n-pad], nil
}
