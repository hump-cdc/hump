package sink

import (
	"reflect"
	"testing"

	"hump/config/decoder"

	"github.com/go-playground/validator/v10"
)

func TestNatsAuthSingleGroupAccepted(t *testing.T) {
	tests := []struct {
		name string
		c    NatsCommon
	}{
		{name: "none", c: NatsCommon{}},
		{name: "credentials", c: NatsCommon{Credentials: mustTemplate(t, "creds")}},
		{name: "nkey", c: NatsCommon{NKeySeed: mustTemplate(t, "seed")}},
		{name: "user and password", c: NatsCommon{Username: mustTemplate(t, "u"), Password: mustTemplate(t, "p")}},
		{name: "password only", c: NatsCommon{Password: mustTemplate(t, "p")}},
		{name: "token", c: NatsCommon{Token: mustTemplate(t, "tok")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.c.URL = mustTemplate(t, "nats://x")
			if err := decoder.Validate.Struct(tt.c); err != nil {
				t.Fatalf("Validate = %v, want nil", err)
			}
		})
	}
}

func TestNatsAuthMultipleGroupsRejected(t *testing.T) {
	c := NatsCommon{
		URL:         mustTemplate(t, "nats://x"),
		Username:    mustTemplate(t, "u"),
		Token:       mustTemplate(t, "tok"),
		Credentials: mustTemplate(t, "creds"),
		NKeySeed:    mustTemplate(t, "seed"),
	}

	if err := decoder.Validate.Struct(c); err == nil {
		t.Fatalf("expected natsauth error for several auth groups")
	}
}

func TestNatsAuthValidatorIgnoresOtherStructs(t *testing.T) {
	sl := &fakeStructLevel{current: reflect.ValueOf(struct{}{})}
	natsAuthValidator(sl)
	if sl.reported {
		t.Fatalf("natsAuthValidator reported an error for a non-NatsCommon struct")
	}
}

// fakeStructLevel feeds natsAuthValidator an arbitrary current value and
// records whether it reported an error.
type fakeStructLevel struct {
	validator.StructLevel
	current  reflect.Value
	reported bool
}

// Current returns the value under validation.
func (f *fakeStructLevel) Current() reflect.Value { return f.current }

// ReportError records that an error was reported.
func (f *fakeStructLevel) ReportError(any, string, string, string, string) { f.reported = true }
