package postgres

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/domain"
)

func TestSnapshotExportParserDecodesRows(t *testing.T) {
	parser, rows := newTestSnapshotExportParser(nil)

	if _, err := parser.Write([]byte("1\talice\n2\t\\N\n")); err != nil {
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

func TestSnapshotExportParserJoinsRowsSplitAcrossWrites(t *testing.T) {
	parser, rows := newTestSnapshotExportParser(nil)

	for _, chunk := range []string{"1\tal", "ice\n2", "\tbob\n"} {
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

	if _, err := parser.Write([]byte("1\ta\\tb\\nc\\\\d\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if got := (*rows)[0]["name"]; got != "a\tb\nc\\d" {
		t.Fatalf("name = %q, want the escapes undone", got)
	}
}

func TestSnapshotExportParserRejectsMalformedRows(t *testing.T) {
	for _, line := range []string{"1\n", "1\ta\tb\n"} {
		parser, _ := newTestSnapshotExportParser(nil)

		if _, err := parser.Write([]byte(line)); !errors.Is(err, errCopyRow) {
			t.Errorf("Write(%q) error = %v, want errCopyRow", line, err)
		}
	}
}

func TestSnapshotExportParserRejectsUndecodableField(t *testing.T) {
	parser, _ := newTestSnapshotExportParser(nil)

	if _, err := parser.Write([]byte("one\talice\n")); err == nil {
		t.Fatal("Write() error = nil, want a decode error")
	}
}

func TestSnapshotExportParserStopsOnRowError(t *testing.T) {
	stop := errors.New("stop")
	parser, rows := newTestSnapshotExportParser(stop)

	if _, err := parser.Write([]byte("1\talice\n2\tbob\n")); !errors.Is(err, stop) {
		t.Fatalf("Write() error = %v, want the onRow error", err)
	}

	if len(*rows) != 1 {
		t.Fatalf("rows = %d, want the stream to stop at the first", len(*rows))
	}
}

func TestSnapshotExportParserFinishReportsUnterminatedRow(t *testing.T) {
	parser, _ := newTestSnapshotExportParser(nil)

	if _, err := parser.Write([]byte("1\talice")); err != nil {
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

// newTestSnapshotExportParser returns a parser over an int4 id and a text name that
// collects every row, answering each with err.
func newTestSnapshotExportParser(err error) (*snapshotExportParser, *[]domain.DbRecord) {
	rows := &[]domain.DbRecord{}
	columns := []snapshotColumn{{name: "id", oid: pgtype.Int4OID}, {name: "name", oid: pgtype.TextOID}}

	return newSnapshotExportParser(columns, pgtype.NewMap(), func(record domain.DbRecord) error {
		*rows = append(*rows, record)
		return err
	}), rows
}
