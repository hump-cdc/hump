package postgres

import (
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/pkg/template"
)

func TestValueDecoderJSONNumbers(t *testing.T) {
	input := `{"integer":9007199254740993,"double":1.234567890123456789,"nested":[1e400,{"n":-9007199254740993}],"bool":true,"null":null}`
	want := map[string]any{
		"integer": json.Number("9007199254740993"), "double": json.Number("1.234567890123456789"),
		"nested": []any{json.Number("1e400"), map[string]any{"n": json.Number("-9007199254740993")}},
		"bool":   true, "null": nil,
	}
	for _, oid := range []uint32{pgtype.JSONOID, pgtype.JSONBOID} {
		t.Run(strconv.FormatUint(uint64(oid), 10), func(t *testing.T) {
			got, err := decodeText(newValueTypeMap(), oid, []byte(input))
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("decodeText() = %#v, %v, want %#v", got, err, want)
			}
			for _, input := range []string{`9007199254740993`, `1.234567890123456789`, `1e400`} {
				got, err := decodeText(newValueTypeMap(), oid, []byte(input+" \n\t"))
				if err != nil || got != json.Number(input) {
					t.Errorf("decodeText(%q) = %#v, %v", input, got, err)
				}
			}
			for _, input := range []string{`{} {}`, `1 true`, `{} trailing`, ``, `{"n":`} {
				if _, err := decodeText(newValueTypeMap(), oid, []byte(input)); err == nil {
					t.Errorf("decodeText(%q) accepted invalid JSON", input)
				}
			}
		})
	}
	for _, oid := range []uint32{pgtype.JSONArrayOID, pgtype.JSONBArrayOID} {
		got, err := decodeText(newValueTypeMap(), oid, []byte(valueSQLArray(input, `[9007199254740993,1.234567890123456789]`)))
		wantArray := []any{want, []any{json.Number("9007199254740993"), json.Number("1.234567890123456789")}}
		if err != nil || !reflect.DeepEqual(got, wantArray) {
			t.Errorf("decodeText(array oid %d) = %#v, %v, want %#v", oid, got, err, wantArray)
		}
		if _, err := decodeText(newValueTypeMap(), oid, []byte(valueSQLArray(`{} {}`))); err == nil {
			t.Errorf("array oid %d accepted trailing JSON", oid)
		}
	}
}

func TestValueTypeMapDoesNotMutatePGXDefaults(t *testing.T) {
	defaultMap := pgtype.NewMap()
	custom := newValueTypeMap()
	for _, pair := range [][2]uint32{{pgtype.JSONOID, pgtype.JSONArrayOID}, {pgtype.JSONBOID, pgtype.JSONBArrayOID}} {
		original, _ := defaultMap.TypeForOID(pair[0])
		element, _ := custom.TypeForOID(pair[0])
		array, _ := custom.TypeForOID(pair[1])
		originalArray, _ := defaultMap.TypeForOID(pair[1])
		if original == element || originalArray == array || array.Codec.(*pgtype.ArrayCodec).ElementType != element {
			t.Fatalf("oid %d: codecs were shared or array element was not rebound", pair[0])
		}
		got, err := decodeText(defaultMap, pair[0], []byte(`{"n":1}`))
		if err != nil || !reflect.DeepEqual(got, map[string]any{"n": float64(1)}) {
			t.Fatalf("pgx default codec changed: %#v, %v", got, err)
		}
	}
}

func TestValueDecoderIntervals(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{"00:00:00", "0"},
		{"00:00:00.000001", "1"},
		{"-00:00:00.000001", "-1"},
		{"1 mon -2 days -03:04:05.000001", "2445954999999"},
		{"-1 mons 2 days 03:04:05.000001", "-2445954999999"},
		{"178956970 years 7 mons 2147483647 days 2562047788:00:54.775807", "5842218454018254775807"},
		{"-178956970 years -8 mons -2147483648 days -2562047788:00:54.775808", "-5842218456734454775808"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := decodeText(newValueTypeMap(), pgtype.IntervalOID, []byte(tt.input))
			if err != nil || got != json.Number(tt.want) {
				t.Fatalf("decodeText() = %#v, %v, want %s", got, err, tt.want)
			}
			encoded, err := json.Marshal(got)
			if err != nil || string(encoded) != tt.want {
				t.Fatalf("Marshal() = %s, %v, want unquoted %s", encoded, err, tt.want)
			}
			array, err := decodeText(newValueTypeMap(), pgtype.IntervalArrayOID, []byte(valueSQLArray(tt.input)))
			if err != nil || !reflect.DeepEqual(array, []any{json.Number(tt.want)}) {
				t.Fatalf("decodeText(array) = %#v, %v", array, err)
			}
		})
	}
}

func TestValueDecoderTemporalAndArrays(t *testing.T) {
	uuid := [16]byte{0x0f, 0x8f, 0xad, 0x5b, 0xd9, 0xcb, 0x46, 0x9f, 0xa1, 0x65, 0x70, 0x86, 0x77, 0x28, 0x95, 0x0e}
	for _, tt := range []struct {
		name  string
		oid   uint32
		input string
		want  any
	}{
		{"timestamp", pgtype.TimestampOID, "2026-03-29 02:30:04.123456", "2026-03-29T02:30:04.123456"},
		{"timestamp whole seconds", pgtype.TimestampOID, "2026-10-25 02:30:00", "2026-10-25T02:30:00"},
		{"timestamp infinity", pgtype.TimestampOID, "infinity", infiniteTime(pgtype.Infinity)},
		{"timestamp negative infinity", pgtype.TimestampOID, "-infinity", infiniteTime(pgtype.NegativeInfinity)},
		{"timestamp array", pgtype.TimestampArrayOID, valueSQLArray("2026-03-29 02:30:04.123456", "infinity", "-infinity"), []any{"2026-03-29T02:30:04.123456", infiniteTime(pgtype.Infinity), infiniteTime(pgtype.NegativeInfinity)}},
		{"date array", pgtype.DateArrayOID, valueSQLArray("2026-10-31", "infinity", "-infinity"), []any{time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), infiniteTime(pgtype.Infinity), infiniteTime(pgtype.NegativeInfinity)}},
		{"uuid array", pgtype.UUIDArrayOID, `{0f8fad5b-d9cb-469f-a165-70867728950e,NULL}`, []any{uuid, nil}},
		{"timestamp null array", pgtype.TimestampArrayOID, `{NULL}`, []any{nil}},
		{"empty array", pgtype.IntervalArrayOID, `{}`, []any{}},
		{"multidimensional shape", pgtype.TimestampArrayOID, `{{"2026-01-01 00:00:00",NULL},{infinity,-infinity}}`, []any{"2026-01-01T00:00:00", nil, infiniteTime(pgtype.Infinity), infiniteTime(pgtype.NegativeInfinity)}},
		{"non-default bounds", pgtype.UUIDArrayOID, `[0:1]={0f8fad5b-d9cb-469f-a165-70867728950e,NULL}`, []any{uuid, nil}},
		{"float infinity", pgtype.Float8ArrayOID, `{Infinity,-Infinity,NULL}`, []any{math.Inf(1), math.Inf(-1), nil}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeText(newValueTypeMap(), tt.oid, []byte(tt.input))
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("decodeText() = %#v, %v, want %#v", got, err, tt.want)
			}
		})
	}
	for _, oid := range []uint32{pgtype.TimestamptzOID, pgtype.TimestamptzArrayOID, pgtype.DateOID} {
		input := "2026-10-31 12:34:56.123456+05:30"
		switch oid {
		case pgtype.DateOID:
			input = "2026-10-31"
		case pgtype.TimestamptzArrayOID:
			input = strings.TrimSuffix(valueSQLArray(input), "}") + ",infinity,-infinity,NULL}"
		}
		defaultType, _ := pgtype.NewMap().TypeForOID(oid)
		want, err := defaultType.Codec.DecodeValue(pgtype.NewMap(), oid, pgtype.TextFormatCode, []byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if array, ok := want.([]any); ok {
			array[1], array[2] = infiniteTime(pgtype.Infinity), infiniteTime(pgtype.NegativeInfinity)
		}
		got, err := decodeText(newValueTypeMap(), oid, []byte(input))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("oid %d changed: %#v, %v, want %#v", oid, got, err, want)
		}
	}
}

func TestValueDecoderPreservesUUIDArrayBaseline(t *testing.T) {
	input := []byte(`{0f8fad5b-d9cb-469f-a165-70867728950e,NULL}`)
	baselineMap := pgtype.NewMap()
	baselineType, _ := baselineMap.TypeForOID(pgtype.UUIDArrayOID)
	want, err := baselineType.Codec.DecodeValue(baselineMap, pgtype.UUIDArrayOID, pgtype.TextFormatCode, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := want.([]any)[0].([16]byte); !ok {
		t.Fatalf("baseline UUID array element = %T, want [16]byte", want.([]any)[0])
	}
	got, err := decodeText(newValueTypeMap(), pgtype.UUIDArrayOID, input)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("decodeText() = %#v, %v, want baseline %#v", got, err, want)
	}
}

func TestValueDecoderArrayInfinityRendering(t *testing.T) {
	for _, oid := range []uint32{pgtype.TimestampArrayOID, pgtype.TimestamptzArrayOID, pgtype.DateArrayOID} {
		value, err := decodeText(newValueTypeMap(), oid, []byte(`{infinity,-infinity,NULL}`))
		if err != nil {
			t.Fatal(err)
		}
		for _, tt := range []struct{ source, want string }{
			{`{{ index . 0 }} {{ index . 1 }}`, "infinity -infinity"},
			{`{{ . | json }}`, `[null,null,null]`},
			{`{{ . | yaml false }}`, "- null\n- null\n- null"},
		} {
			tmpl, err := template.New(tt.source)
			if err != nil {
				t.Fatal(err)
			}
			got, err := tmpl.Render(value)
			if err != nil || got != tt.want {
				t.Errorf("oid %d Render(%q) = %q, %v, want %q", oid, tt.source, got, err, tt.want)
			}
		}
	}
}

func TestValueDecoderSnapshotWALParityAndTOAST(t *testing.T) {
	columns := []snapshotColumn{
		{"json", pgtype.JSONOID}, {"jsonb", pgtype.JSONBOID}, {"interval", pgtype.IntervalOID},
		{"timestamp", pgtype.TimestampOID}, {"tstz", pgtype.TimestamptzOID}, {"json_array", pgtype.JSONArrayOID},
		{"jsonb_array", pgtype.JSONBArrayOID}, {"interval_array", pgtype.IntervalArrayOID},
		{"timestamp_array", pgtype.TimestampArrayOID}, {"tstz_array", pgtype.TimestamptzArrayOID},
		{"date_array", pgtype.DateArrayOID}, {"uuid_array", pgtype.UUIDArrayOID}, {"null", pgtype.JSONOID}, {"unknown", 999999},
	}
	values := []string{
		`{"nested":[9007199254740993,1.234567890123456789]}`, `[1e400,null]`, "178956970 years 7 mons", "2026-03-29 02:30:04.123456",
		"2026-10-31 12:34:56.123456+00", valueSQLArray(`[9007199254740993]`), valueSQLArray(`{"n":1.234567890123456789}`),
		valueSQLArray("-1 mons 2 days 03:04:05.000001"), `{infinity,-infinity,NULL}`, `{infinity,-infinity,NULL}`,
		`{infinity,-infinity,NULL}`, `{0f8fad5b-d9cb-469f-a165-70867728950e,NULL}`, "", "unregistered",
	}
	relation := &pglogrepl.RelationMessageV2{}
	old, unchanged := &pglogrepl.TupleData{}, &pglogrepl.TupleData{}
	fields := []string{"(12,34)"}
	for i, column := range columns {
		relation.Columns = append(relation.Columns, &pglogrepl.RelationMessageColumn{Name: column.name, DataType: column.oid})
		if column.name == "null" {
			old.Columns = append(old.Columns, &pglogrepl.TupleDataColumn{DataType: 'n'})
			fields = append(fields, `\N`)
		} else {
			old.Columns = append(old.Columns, &pglogrepl.TupleDataColumn{DataType: 't', Data: []byte(values[i])})
			fields = append(fields, escapeCopyText(values[i]))
		}
		unchanged.Columns = append(unchanged.Columns, &pglogrepl.TupleDataColumn{DataType: 'u'})
	}
	w := &wal{typeMap: newValueTypeMap()}
	prev, next, err := w.decodeRow(old, pglogrepl.UpdateMessageTupleTypeOld, unchanged, relation)
	if err != nil || !reflect.DeepEqual(prev, next) {
		t.Fatalf("TOAST reconstruction = %#v, %#v, %v", prev, next, err)
	}
	uuid := [16]byte{0x0f, 0x8f, 0xad, 0x5b, 0xd9, 0xcb, 0x46, 0x9f, 0xa1, 0x65, 0x70, 0x86, 0x77, 0x28, 0x95, 0x0e}
	if !reflect.DeepEqual(next["uuid_array"], []any{uuid, nil}) {
		t.Fatalf("WAL UUID array = %#v, want raw UUID bytes and NULL", next["uuid_array"])
	}
	data := strings.Join(fields, "\t") + "\n"
	for size := 1; size <= len(data); size++ {
		var rows []domain.DbRecord
		parser := newSnapshotExportParser(usersTable(), columns, newValueTypeMap(), func(record domain.DbRecord, _ database.SnapshotRowMetadata) error {
			rows = append(rows, record)
			return nil
		})
		for _, chunk := range chunkBytes([]byte(data), size) {
			if _, err := parser.Write(chunk); err != nil {
				t.Fatalf("chunk %d: %v", size, err)
			}
			for i := range chunk {
				chunk[i] = 0xff
			}
		}
		if err := parser.finish(); err != nil || !reflect.DeepEqual(rows, []domain.DbRecord{next}) {
			t.Fatalf("chunk %d: snapshot = %#v, %v, want WAL %#v", size, rows, err, next)
		}
	}
}

// valueSQLArray quotes elements without changing JSON's own nested array structure.
func valueSQLArray(elements ...string) string {
	quoted := make([]string, len(elements))
	for i, element := range elements {
		quoted[i] = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(element) + `"`
	}
	return "{" + strings.Join(quoted, ",") + "}"
}
