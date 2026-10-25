package sink

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"hump/internal/domain"
	sinkport "hump/internal/port/sink"
	"hump/pkg/template"
)

func TestDefaultTemplatesWriteNonFiniteFloatsAsNull(t *testing.T) {
	ctx := &sinkport.Context{
		Timestamp: time.Unix(1700000000, 0),
		Operation: domain.OperationTypeInsert,
		Schema:    "public",
		Table:     "readings",
		Next:      domain.DbRecord{"nan": math.NaN(), "inf": math.Inf(1), "ninf": float32(math.Inf(-1)), "ok": 1.5},
	}

	rendered, err := mustTemplate(t, DefaultPayload).Render(ctx)
	if err != nil {
		t.Fatalf("Render(DefaultPayload) error = %v", err)
	}

	var got defaultPayloadEnvelope
	if err := json.Unmarshal([]byte(rendered), &got); err != nil {
		t.Fatalf("Render(DefaultPayload) = %q, not JSON: %v", rendered, err)
	}

	if want := `{"inf":null,"nan":null,"ninf":null,"ok":1.5}`; string(got.After) != want {
		t.Errorf("after = %s, want %s", got.After, want)
	}

	if _, err := mustTemplate(t, DefaultIdempotencyKey).Render(ctx); err != nil {
		t.Errorf("Render(DefaultIdempotencyKey) error = %v", err)
	}
}

func FuzzDefaultPayload(f *testing.F) {
	f.Add("public", "users", "name", "alice", int64(7), 1.5, true, uint8(0), int64(1700000000000000000))
	f.Add("", "", "", "", int64(0), 0.0, false, uint8(1), int64(0))
	f.Add("Schema\"x", "t\\ab\nle", "<script>", "\u2028&\x00", int64(math.MinInt64), math.Copysign(0, -1), false, uint8(2), int64(-1))
	f.Add("s", "t", "n", "\xff\xfe", int64(math.MaxInt64), math.MaxFloat64, true, uint8(3), int64(math.MaxInt64))
	f.Add("public", "readings", "value", "x", int64(1), math.NaN(), false, uint8(0), int64(0))
	f.Add("public", "readings", "value", "x", int64(1), math.Inf(1), false, uint8(1), int64(0))
	f.Add("public", "readings", "value", "x", int64(1), math.Inf(-1), false, uint8(2), int64(0))

	payload := mustDynamicTemplate(f, DefaultPayload)
	idempotencyKey := mustDynamicTemplate(f, DefaultIdempotencyKey)

	f.Fuzz(func(t *testing.T, schema, table, column, text string, number int64, fraction float64, flag bool, op uint8, nanos int64) {
		operation := domain.OperationTypes[int(op)%len(domain.OperationTypes)]
		record := domain.DbRecord{
			"n": number, "f": fraction, "f4": float32(fraction), "b": flag, "z": nil, "u": domain.Unavailable,
		}
		record[column] = text

		ctx := &sinkport.Context{
			Version:   "1.0.0",
			Timestamp: time.Unix(0, nanos),
			Operation: operation,
			Database:  "db",
			Schema:    domain.TableSchema(schema),
			Table:     domain.TableName(table),
		}
		switch operation {
		case domain.OperationTypeInsert:
			ctx.Next = record
		case domain.OperationTypeUpdate:
			ctx.Prev, ctx.Next = record, record
		case domain.OperationTypeDelete:
			ctx.Prev = record
		default:
		}

		rendered, err := payload.Render(ctx)
		if err != nil {
			t.Fatalf("Render(DefaultPayload) error = %v", err)
		}

		var got defaultPayloadEnvelope
		if err := json.Unmarshal([]byte(rendered), &got); err != nil {
			t.Fatalf("Render(DefaultPayload) = %q, not JSON: %v", rendered, err)
		}

		if want := defaultPayloadOps[operation]; got.Op != want {
			t.Errorf("op = %q, want %q for %s", got.Op, want, operation)
		}

		if utf8.ValidString(schema) && utf8.ValidString(table) && (got.Source.Schema != schema || got.Source.Table != table) {
			t.Errorf("source = %q.%q, want %q.%q", got.Source.Schema, got.Source.Table, schema, table)
		}

		if want := ctx.Timestamp.Unix() * 1000; got.Source.TsMs != want {
			t.Errorf("source.ts_ms = %d, want %d", got.Source.TsMs, want)
		}

		assertDefaultPayloadRecord(t, "before", got.Before, ctx.Prev)
		assertDefaultPayloadRecord(t, "after", got.After, ctx.Next)

		key, err := idempotencyKey.Render(ctx)
		if err != nil {
			t.Fatalf("Render(DefaultIdempotencyKey) error = %v", err)
		}

		if len(key) != 64 || strings.Trim(key, "0123456789abcdef") != "" {
			t.Errorf("idempotency key = %q, want 64 lowercase hex digits", key)
		}

		if again, err := idempotencyKey.Render(ctx); err != nil || again != key {
			t.Errorf("idempotency key = %q, then %q, %v, want it stable for the same row", key, again, err)
		}
	})
}

// defaultPayloadOps is the Debezium op letter DefaultPayload writes for each
// operation.
var defaultPayloadOps = map[domain.OperationType]string{
	domain.OperationTypeInsert:   "c",
	domain.OperationTypeUpdate:   "u",
	domain.OperationTypeDelete:   "d",
	domain.OperationTypeTruncate: "t",
}

// defaultPayloadEnvelope is the part of the DefaultPayload envelope the fuzz
// target checks. The records stay raw so int64 values keep their precision.
type defaultPayloadEnvelope struct {
	Before json.RawMessage `json:"before"`
	After  json.RawMessage `json:"after"`
	Source struct {
		Schema string `json:"schema"`
		Table  string `json:"table"`
		TsMs   int64  `json:"ts_ms"`
	} `json:"source"`
	Op string `json:"op"`
}

// mustDynamicTemplate parses source as a dynamic template or fails the test.
func mustDynamicTemplate(tb testing.TB, source string) *template.Template {
	tb.Helper()

	tmpl, err := template.New(source)
	if err != nil || !tmpl.IsDynamic() {
		tb.Fatalf("template.New(%q) = dynamic %t, %v, want a dynamic template", source, tmpl.IsDynamic(), err)
	}

	return tmpl
}

// assertDefaultPayloadRecord checks that the envelope field name holds record
// as JSON, with NaN and ±Inf floats as null, or is absent when there is no
// record.
func assertDefaultPayloadRecord(t *testing.T, name string, raw json.RawMessage, record domain.DbRecord) {
	t.Helper()

	if record == nil {
		if raw != nil {
			t.Errorf("%s = %s, want it absent", name, raw)
		}

		return
	}

	finite := make(domain.DbRecord, len(record))
	for column, value := range record {
		switch v := value.(type) {
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				value = nil
			}
		case float32:
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				value = nil
			}
		}
		finite[column] = value
	}

	want, err := json.Marshal(finite)
	if err != nil {
		t.Fatalf("json.Marshal(%v) error = %v", record, err)
	}

	var got bytes.Buffer
	if err := json.Compact(&got, raw); err != nil || !bytes.Equal(got.Bytes(), want) {
		t.Errorf("%s = %s, want %s", name, raw, want)
	}
}
