package config

import (
	"slices"
	"testing"

	"hump/internal/domain"
)

func ops(values ...domain.OperationType) []domain.OperationType { return values }

// sinkOps returns the operations of every sink of the table named name, in order.
func sinkOps(t *testing.T, cfg Configuration, name domain.TableName) [][]domain.OperationType {
	t.Helper()

	for _, table := range cfg.Tables {
		if table.Name != name {
			continue
		}

		out := make([][]domain.OperationType, 0, len(table.Sinks))
		for _, sc := range table.Sinks {
			out = append(out, sc.Operations)
		}
		return out
	}

	t.Fatalf("table %q not found", name)
	return nil
}

func assertOps(t *testing.T, got [][]domain.OperationType, want ...[]domain.OperationType) {
	t.Helper()

	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("operations: want %v, got %v", want, got)
	}
}

func TestParse_OperationsDefaultToAll(t *testing.T) {
	cfg, err := Parse(`tables:
  - table: events
    sinks:
      - http: { url: http://x }
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	assertOps(t, sinkOps(t, cfg, "events"), domain.OperationTypes)
}

func TestParse_OperationsOnSingleTable(t *testing.T) {
	cfg, err := Parse(`tables:
  - table: events
    operations: [insert, delete]
    sinks:
      - http: { url: http://x }
      - http: { url: http://y }
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := ops(domain.OperationTypeInsert, domain.OperationTypeDelete)
	assertOps(t, sinkOps(t, cfg, "events"), want, want)
}

func TestParse_OperationsOnTableGroup(t *testing.T) {
	cfg, err := Parse(`tables:
  - tables:
      - table: posts
      - table: comments
    operations: [update]
    sinks:
      - http: { url: http://x }
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := ops(domain.OperationTypeUpdate)
	assertOps(t, sinkOps(t, cfg, "posts"), want)
	assertOps(t, sinkOps(t, cfg, "comments"), want)
}

func TestParse_TableOperationsOverrideSinkAndPreset(t *testing.T) {
	cfg, err := Parse(`sinks:
  - name: h1
    http: { url: http://x }
    operations: [delete]
tables:
  - table: events
    operations: [insert]
    sinks:
      - preset: h1
      - http: { url: http://y }
        operations: [truncate]
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := ops(domain.OperationTypeInsert)
	assertOps(t, sinkOps(t, cfg, "events"), want, want)
}

func TestParse_SinkOperationsApplyWithoutTableOperations(t *testing.T) {
	cfg, err := Parse(`sinks:
  - name: h1
    http: { url: http://x }
    operations: [delete]
tables:
  - table: events
    sinks:
      - preset: h1
      - http: { url: http://y }
        operations: [truncate]
      - http: { url: http://z }
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	assertOps(t, sinkOps(t, cfg, "events"),
		ops(domain.OperationTypeDelete),
		ops(domain.OperationTypeTruncate),
		domain.OperationTypes,
	)
}

func TestParse_OperationsSurviveTableMerge(t *testing.T) {
	cfg, err := Parse(`tables:
  - table: events
    operations: [insert]
    sinks:
      - http: { url: http://x }
  - tables:
      - table: posts
      - table: events
    operations: [update, delete]
    sinks:
      - http: { url: http://y }
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	assertOps(t, sinkOps(t, cfg, "events"),
		ops(domain.OperationTypeInsert),
		ops(domain.OperationTypeUpdate, domain.OperationTypeDelete),
	)
	assertOps(t, sinkOps(t, cfg, "posts"), ops(domain.OperationTypeUpdate, domain.OperationTypeDelete))
}

func TestParse_InvalidOperationsRejected(t *testing.T) {
	for name, value := range map[string]string{
		"unknown":   "[bogus]",
		"empty":     "[]",
		"duplicate": "[insert, insert]",
	} {
		t.Run(name, func(t *testing.T) {
			src := `tables:
  - table: events
    operations: ` + value + `
    sinks:
      - http: { url: http://x }
`
			if _, err := Parse(src); err == nil {
				t.Fatalf("expected error for operations %s", value)
			}
		})
	}
}
