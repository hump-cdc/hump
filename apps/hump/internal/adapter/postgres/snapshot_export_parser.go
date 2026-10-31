package postgres

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/domain"
	"hump/internal/port/database"
)

// errCopyRow indicates a COPY text row that does not match the selected columns.
var errCopyRow = errors.New("malformed COPY row")

// snapshotExportParser separates the leading physical CTID from user columns
// and combines it with the leaf identity supplied by the snapshot layout.
type snapshotExportParser struct {
	leaf    domain.TableIdentifier
	columns []snapshotColumn
	typeMap *pgtype.Map
	onRow   func(domain.DbRecord, database.SnapshotRowMetadata) error
	partial []byte // partial is the start of a row whose end has not arrived yet.
}

// newSnapshotExportParser returns a parser decoding rows of columns with m.
func newSnapshotExportParser(leaf domain.TableIdentifier, columns []snapshotColumn, m *pgtype.Map, onRow func(domain.DbRecord, database.SnapshotRowMetadata) error) *snapshotExportParser {
	return &snapshotExportParser{leaf: leaf, columns: columns, typeMap: m, onRow: onRow}
}

// Write consumes data, which may hold any number of rows and need not end on a
// row boundary, calling onRow for every complete row. A row or onRow error
// stops the stream.
func (p *snapshotExportParser) Write(data []byte) (int, error) {
	n := len(data)

	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		if end < 0 {
			// pgconn reuses its read buffer, so the tail is copied out.
			p.partial = append(p.partial, data...)
			break
		}

		line := data[:end]
		if len(p.partial) > 0 {
			line = append(p.partial, line...)
			p.partial = p.partial[:0]
		}

		if err := p.row(line); err != nil {
			return 0, err
		}

		data = data[end+1:]
	}

	return n, nil
}

// finish reports a row left unterminated by the end of the stream and resets
// the parser for the next one.
func (p *snapshotExportParser) finish() error {
	if len(p.partial) > 0 {
		p.partial = p.partial[:0]
		return fmt.Errorf("unterminated row: %w", errCopyRow)
	}

	return nil
}

// row decodes one row and hands it to onRow.
func (p *snapshotExportParser) row(line []byte) error {
	record := make(domain.DbRecord, len(p.columns))
	metadata := database.SnapshotRowMetadata{Schema: p.leaf.Schema, Table: p.leaf.Name}
	fields := len(p.columns) + 1

	for i := range fields {
		field := line
		if end := bytes.IndexByte(line, '\t'); end >= 0 {
			field, line = line[:end], line[end+1:]
		} else {
			line = nil
		}

		if i == fields-1 && line != nil {
			return fmt.Errorf("more fields than %d columns: %w", fields, errCopyRow)
		}
		if i < fields-1 && line == nil {
			return fmt.Errorf("%d fields, expected %d: %w", i+1, fields, errCopyRow)
		}

		if i == 0 {
			if len(field) == 0 || string(field) == `\N` {
				return fmt.Errorf("missing physical identity field %d: %w", i+1, errCopyRow)
			}
			metadata.CTID = string(p.unescapeCopyText(field))
			continue
		}

		column := p.columns[i-1]
		if string(field) == `\N` {
			record[column.name] = nil
			continue
		}

		value, err := decodeText(p.typeMap, column.oid, p.unescapeCopyText(field))
		if err != nil {
			return fmt.Errorf("decode column %q (oid %d): %w", column.name, column.oid, err)
		}
		record[column.name] = value
	}

	return p.onRow(record, metadata)
}

// unescapeCopyText undoes the backslash escapes of the COPY text format:
// \b \f \n \r \t \v, \ooo octal, \xhh hex, and any other escaped byte as itself.
func (p *snapshotExportParser) unescapeCopyText(field []byte) []byte {
	if bytes.IndexByte(field, '\\') < 0 {
		return field
	}

	out := make([]byte, 0, len(field))
	for i := 0; i < len(field); i++ {
		c := field[i]
		if c != '\\' || i == len(field)-1 {
			out = append(out, c)
			continue
		}

		i++
		c = field[i]

		switch {
		case c == 'b':
			out = append(out, '\b')
		case c == 'f':
			out = append(out, '\f')
		case c == 'n':
			out = append(out, '\n')
		case c == 'r':
			out = append(out, '\r')
		case c == 't':
			out = append(out, '\t')
		case c == 'v':
			out = append(out, '\v')
		case c >= '0' && c <= '7':
			value := c - '0'
			for j := 0; j < 2 && i+1 < len(field) && field[i+1] >= '0' && field[i+1] <= '7'; j++ {
				i++
				value = value<<3 | (field[i] - '0')
			}
			out = append(out, value)
		case c == 'x' && i+1 < len(field) && p.isHexDigit(field[i+1]):
			i++
			value := p.hexValue(field[i])
			if i+1 < len(field) && p.isHexDigit(field[i+1]) {
				i++
				value = value<<4 | p.hexValue(field[i])
			}
			out = append(out, value)
		default:
			out = append(out, c)
		}
	}

	return out
}

// isHexDigit reports whether c is a hexadecimal digit.
func (p *snapshotExportParser) isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// hexValue returns the value of the hexadecimal digit c.
func (p *snapshotExportParser) hexValue(c byte) byte {
	switch {
	case c >= 'a':
		return c - 'a' + 10
	case c >= 'A':
		return c - 'A' + 10
	default:
		return c - '0'
	}
}
