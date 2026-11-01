package startsnapshot

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/snapshot"
)

func TestStartResolvesBatchWithoutMutatingInput(t *testing.T) {
	runner := &fakeRunner{}
	input := []snapshot.Request{
		{Table: domain.TableIdentifier{Name: "users"}, Where: "id > 1"},
		{Table: domain.TableIdentifier{Name: "orders"}},
	}
	original := slices.Clone(input)
	if err := NewUseCase(testConfiguration(), runner).Start(input); err != nil {
		t.Fatal(err)
	}
	want := []snapshot.Request{
		{Table: domain.TableIdentifier{Schema: "public", Name: "users"}, Where: "id > 1"},
		{Table: domain.TableIdentifier{Schema: "sales", Name: "orders"}},
	}
	if runner.calls != 1 || !reflect.DeepEqual(runner.started, want) {
		t.Fatalf("Start calls = %d, tables = %+v; want one call with %+v", runner.calls, runner.started, want)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("input mutated: %+v", input)
	}
}

func TestStartRejectsEntireBatchWithoutMutatingInput(t *testing.T) {
	for _, denied := range []domain.TableIdentifier{
		{Schema: "private", Name: "users"},
		{Schema: "pg_catalog", Name: "pg_authid"},
		{Schema: "public", Name: "users_2026"},
		{Name: "missing"},
	} {
		t.Run(string(denied.Schema)+"."+string(denied.Name), func(t *testing.T) {
			runner := &fakeRunner{}
			input := []snapshot.Request{{Table: domain.TableIdentifier{Name: "users"}}, {Table: denied}}
			original := slices.Clone(input)
			if err := NewUseCase(testConfiguration(), runner).Start(input); !errors.Is(err, ErrTableNotConfigured) {
				t.Fatalf("Start = %v, want ErrTableNotConfigured", err)
			}
			if runner.calls != 0 || !reflect.DeepEqual(input, original) {
				t.Fatalf("calls = %d, input = %+v; want no calls and unchanged %+v", runner.calls, input, original)
			}
		})
	}
}

// testConfiguration authorizes the two test targets with explicit schemas.
func testConfiguration() config.Configuration {
	return config.Configuration{Tables: []config.Table{{Schema: "public", Name: "users"}, {Schema: "sales", Name: "orders"}}}
}
