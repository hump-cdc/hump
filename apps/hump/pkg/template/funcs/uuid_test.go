package funcs

import (
	"errors"
	"regexp"
	"testing"
)

var uuidV4Re = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestUUIDv4Template(t *testing.T) {
	seen := make(map[string]struct{}, 32)
	for range 32 {
		got := mustRender(t, `{{ uuidv4 }}`, nil)
		if !uuidV4Re.MatchString(got) {
			t.Fatalf("not v4 form: %q", got)
		}
		if _, dup := seen[got]; dup {
			t.Fatalf("duplicate uuid: %q", got)
		}
		seen[got] = struct{}{}
	}
}

func TestUUIDv4TemplatePropagatesRandError(t *testing.T) {
	withFailingRand(t)

	if _, err := render(t, `{{ uuidv4 }}`, nil); !errors.Is(err, errRandFailed) {
		t.Fatalf("uuidv4 error = %v, want errRandFailed", err)
	}
}
