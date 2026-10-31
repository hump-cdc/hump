package postgres

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestSnapshotExportParserDecodesRows(t *testing.T) {
	parser, rows := newTestSnapshotExportParser(nil)

	if _, err := parser.Write([]byte("(0,1)\t1\talice\n(0,2)\t2\t\\N\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	want := []domain.DbRecord{
		{"id": int32(1), "name": "alice"},
		{"id": int32(2), "name": nil},
	}
	if !reflect.DeepEqual(*rows, want) {
		t.Fatalf("rows = %#v, want %#v", *rows, want)
	}
}

func TestSnapshotExportParserSeparatesPhysicalIdentityFromUserColumns(t *testing.T) {
	columns := []snapshotColumn{
		{name: "ctid", oid: pgtype.TextOID},
		{name: "schema", oid: pgtype.TextOID},
		{name: "table", oid: pgtype.TextOID},
		{name: "CTID", oid: pgtype.TextOID},
		{name: "snapshot_row", oid: pgtype.TextOID},
		{name: "tableoid", oid: pgtype.TextOID},
	}
	var rows []domain.DbRecord
	var identities []database.SnapshotRowMetadata
	leaf := domain.TableIdentifier{Schema: "physical\tschema", Name: "leaf\nname"}
	parser := newSnapshotExportParser(leaf, columns, newValueTypeMap(), func(record domain.DbRecord, metadata database.SnapshotRowMetadata) error {
		rows = append(rows, record)
		identities = append(identities, metadata)
		return nil
	})
	data := []byte("(12,34)\tuser-ctid\tuser-schema\tuser-table\tuser-CTID\tuser-alias\tuser-tableoid\n")
	for _, chunk := range chunkBytes(data, 7) {
		if _, err := parser.Write(chunk); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		for i := range chunk {
			chunk[i] = 0xff
		}
	}
	if err := parser.finish(); err != nil {
		t.Fatalf("finish() error = %v", err)
	}
	wantRows := []domain.DbRecord{{"ctid": "user-ctid", "schema": "user-schema", "table": "user-table", "CTID": "user-CTID", "snapshot_row": "user-alias", "tableoid": "user-tableoid"}}
	if !reflect.DeepEqual(rows, wantRows) {
		t.Fatalf("rows = %#v, want %#v", rows, wantRows)
	}
	wantIdentities := []database.SnapshotRowMetadata{{Schema: "physical\tschema", Table: "leaf\nname", CTID: "(12,34)"}}
	if !reflect.DeepEqual(identities, wantIdentities) {
		t.Fatalf("identities = %#v, want %#v", identities, wantIdentities)
	}
}

func TestSnapshotExportParserJoinsRowsSplitAcrossWrites(t *testing.T) {
	parser, rows := newTestSnapshotExportParser(nil)

	for _, chunk := range []string{"(0,", "1)\t1\tal", "ice\n(0,2)\t2", "\tbob\n"} {
		if _, err := parser.Write([]byte(chunk)); err != nil {
			t.Fatalf("Write(%q) error = %v", chunk, err)
		}
	}

	if err := parser.finish(); err != nil {
		t.Fatalf("finish() error = %v", err)
	}

	want := []domain.DbRecord{
		{"id": int32(1), "name": "alice"},
		{"id": int32(2), "name": "bob"},
	}
	if !reflect.DeepEqual(*rows, want) {
		t.Fatalf("rows = %#v, want %#v", *rows, want)
	}
}

func TestSnapshotExportParserUnescapesFields(t *testing.T) {
	parser, rows := newTestSnapshotExportParser(nil)

	if _, err := parser.Write([]byte("(0,1)\t1\ta\\tb\\nc\\\\d\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if got := (*rows)[0]["name"]; got != "a\tb\nc\\d" {
		t.Fatalf("name = %q, want the escapes undone", got)
	}
}

func TestSnapshotExportParserRejectsMalformedRows(t *testing.T) {
	for _, line := range []string{"1\n", "(0,1)\t1\n", "(0,1)\t1\ta\tb\n", "\\N\t1\ta\n", "\t1\ta\n"} {
		parser, _ := newTestSnapshotExportParser(nil)

		if _, err := parser.Write([]byte(line)); !errors.Is(err, errCopyRow) {
			t.Errorf("Write(%q) error = %v, want errCopyRow", line, err)
		}
	}
}

func TestSnapshotExportParserRejectsUndecodableField(t *testing.T) {
	parser, _ := newTestSnapshotExportParser(nil)

	if _, err := parser.Write([]byte("(0,1)\tone\talice\n")); err == nil {
		t.Fatal("Write() error = nil, want a decode error")
	}
}

func TestSnapshotExportParserStopsOnRowError(t *testing.T) {
	stop := errors.New("stop")
	parser, rows := newTestSnapshotExportParser(stop)

	if _, err := parser.Write([]byte("(0,1)\t1\talice\n(0,2)\t2\tbob\n")); !errors.Is(err, stop) {
		t.Fatalf("Write() error = %v, want the onRow error", err)
	}

	if len(*rows) != 1 {
		t.Fatalf("rows = %d, want the stream to stop at the first", len(*rows))
	}
}

func TestSnapshotExportParserFinishReportsUnterminatedRow(t *testing.T) {
	parser, _ := newTestSnapshotExportParser(nil)

	if _, err := parser.Write([]byte("(0,1)\t1\talice")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := parser.finish(); !errors.Is(err, errCopyRow) {
		t.Fatalf("finish() error = %v, want errCopyRow", err)
	}

	if err := parser.finish(); err != nil {
		t.Fatalf("second finish() error = %v, want the parser reset", err)
	}
}

func TestUnescapeCopyText(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`plain`, "plain"},
		{`\b\f\n\r\t\v`, "\b\f\n\r\t\v"},
		{`\\x00ff`, `\x00ff`},
		{`\101\0`, "A\x00"},
		{`\x41\x4`, "A\x04"},
		{`\x6a\x4A`, "jJ"},
		{`\q`, "q"},
		{`\x`, "x"},
		{`end\`, `end\`},
	}

	for _, tt := range tests {
		if got := string((&snapshotExportParser{}).unescapeCopyText([]byte(tt.in))); got != tt.want {
			t.Errorf("unescapeCopyText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func FuzzSnapshotExportParserChunking(f *testing.F) {
	f.Add([]byte("(0,1)\t1\talice\n(0,2)\t2\t\\N\n"), uint8(1))
	f.Add([]byte("(0,1)\t1\ta\\tb\\nc\\\\d\n"), uint8(3))
	f.Add([]byte("1\n1\ta\tb\n"), uint8(2))
	f.Add([]byte("1\talice"), uint8(4))
	f.Add([]byte("\\101\t\\x4\n\n\t\n"), uint8(0))

	typeMap := newValueTypeMap()

	f.Fuzz(func(t *testing.T, data []byte, chunk uint8) {
		whole := collectSnapshotExport(t, typeMap, [][]byte{data})
		chunked := collectSnapshotExport(t, typeMap, chunkBytes(data, int(chunk)%16+1))

		if !reflect.DeepEqual(whole, chunked) {
			t.Fatalf("chunked parse = %#v, want it to match the whole parse %#v", chunked, whole)
		}
	})
}

func FuzzSnapshotExportParserRoundTrip(f *testing.F) {
	f.Add("1", "alice")
	f.Add("", `\N`)
	f.Add("a\tb", "c\nd\re\\f")
	f.Add("\b\f\v", "\x00\xff")

	typeMap := newValueTypeMap()

	f.Fuzz(func(t *testing.T, id, name string) {
		line := "(0,1)\t" + escapeCopyText(id) + "\t" + escapeCopyText(name) + "\n"

		got := collectSnapshotExport(t, typeMap, [][]byte{[]byte(line)})

		want := snapshotExportResult{rows: []domain.DbRecord{{"id": id, "name": name}}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("parse of %q = %#v, want %#v", line, got, want)
		}
	})
}

func FuzzUnescapeCopyText(f *testing.F) {
	f.Add([]byte(`plain`))
	f.Add([]byte(`\b\f\n\r\t\v`))
	f.Add([]byte(`\101\0\777`))
	f.Add([]byte(`\x41\x4\xZZ`))
	f.Add([]byte(`end\`))

	f.Fuzz(func(t *testing.T, field []byte) {
		in := bytes.Clone(field)
		got := (&snapshotExportParser{}).unescapeCopyText(field)

		if !bytes.Equal(field, in) {
			t.Fatalf("unescapeCopyText modified its input: %q, was %q", field, in)
		}

		if len(got) > len(field) {
			t.Fatalf("unescapeCopyText(%q) = %q, longer than its input", field, got)
		}

		if bytes.IndexByte(field, '\\') < 0 && !bytes.Equal(got, field) {
			t.Fatalf("unescapeCopyText(%q) = %q, want a field without escapes unchanged", field, got)
		}
	})
}

// newTestSnapshotExportParser returns a parser over an int4 id and a text name that
// collects every row, answering each with err.
func newTestSnapshotExportParser(err error) (*snapshotExportParser, *[]domain.DbRecord) {
	rows := &[]domain.DbRecord{}
	columns := []snapshotColumn{{name: "id", oid: pgtype.Int4OID}, {name: "name", oid: pgtype.TextOID}}

	return newSnapshotExportParser(usersTable(), columns, newValueTypeMap(), func(record domain.DbRecord, _ database.SnapshotRowMetadata) error {
		*rows = append(*rows, record)
		return err
	}), rows
}

// snapshotExportResult is what a snapshot export parse produced: the rows
// handed to onRow and the error that ended it, as text.
type snapshotExportResult struct {
	rows []domain.DbRecord
	err  string
}

// snapshotExportTextColumns are two text columns, whose codec accepts any
// field, so a parse can fail only on the shape of a row.
var snapshotExportTextColumns = []snapshotColumn{{name: "id", oid: pgtype.TextOID}, {name: "name", oid: pgtype.TextOID}}

// collectSnapshotExport writes chunks to a parser over snapshotExportTextColumns
// the way pgconn does, scribbling over each chunk once Write returns since
// pgconn reuses its read buffer, and then finishes the stream.
func collectSnapshotExport(t *testing.T, m *pgtype.Map, chunks [][]byte) snapshotExportResult {
	t.Helper()

	var result snapshotExportResult
	parser := newSnapshotExportParser(usersTable(), snapshotExportTextColumns, m, func(record domain.DbRecord, _ database.SnapshotRowMetadata) error {
		result.rows = append(result.rows, record)
		return nil
	})

	for _, chunk := range chunks {
		buffer := bytes.Clone(chunk)

		n, err := parser.Write(buffer)
		if err != nil {
			result.err = err.Error()
			return result
		}
		if n != len(buffer) {
			t.Fatalf("Write() = %d, want %d", n, len(buffer))
		}

		for i := range buffer {
			buffer[i] = 0xff
		}
	}

	if err := parser.finish(); err != nil {
		result.err = err.Error()
	}

	return result
}

// chunkBytes splits data into consecutive chunks of size bytes, the last one
// possibly shorter.
func chunkBytes(data []byte, size int) [][]byte {
	var chunks [][]byte
	for len(data) > size {
		chunks = append(chunks, data[:size])
		data = data[size:]
	}

	return append(chunks, data)
}

// escapeCopyText escapes s the way COPY ... TO STDOUT writes a text field.
func escapeCopyText(s string) string {
	return strings.NewReplacer(
		"\\", "\\\\", "\b", `\b`, "\f", `\f`, "\n", `\n`, "\r", `\r`, "\t", `\t`, "\v", `\v`,
	).Replace(s)
}
