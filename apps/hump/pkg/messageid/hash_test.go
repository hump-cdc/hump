package messageid

import "testing"

func TestHashSeparatesComponents(t *testing.T) {
	values := [][]string{{"ab", "c"}, {"a", "bc"}, {"a", "", "bc"}, {"a\x00", "bc"}, {"snapshot", "a", "bc"}, {"WAL", "a", "bc"}}
	seen := make(map[string]bool)
	for _, parts := range values {
		id := Hash(parts...)
		if len(id) != 64 || seen[id] || id != Hash(parts...) {
			t.Fatalf("Hash(%q) = %q: duplicate, unstable or invalid digest", parts, id)
		}
		seen[id] = true
	}
}
