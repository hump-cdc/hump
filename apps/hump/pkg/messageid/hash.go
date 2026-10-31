// Package messageid hashes unambiguous source-event identity components.
package messageid

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// Hash returns a hexadecimal SHA256 digest of length-prefixed components.
func Hash(parts ...string) string {
	h := sha256.New()
	var size [8]byte
	for _, part := range parts {
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		_, _ = h.Write(size[:])
		_, _ = h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}
