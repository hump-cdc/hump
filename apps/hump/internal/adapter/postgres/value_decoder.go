package postgres

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// newValueTypeMap installs private JSON codecs, including the array element types.
func newValueTypeMap() *pgtype.Map {
	m := pgtype.NewMap()
	for _, entry := range []struct {
		oid      uint32
		arrayOID uint32
		codec    pgtype.Codec
	}{
		{pgtype.JSONOID, pgtype.JSONArrayOID, &pgtype.JSONCodec{Marshal: json.Marshal, Unmarshal: unmarshalJSONNumber}},
		{pgtype.JSONBOID, pgtype.JSONBArrayOID, &pgtype.JSONBCodec{Marshal: json.Marshal, Unmarshal: unmarshalJSONNumber}},
	} {
		original, _ := m.TypeForOID(entry.oid)
		element := &pgtype.Type{Name: original.Name, OID: entry.oid, Codec: entry.codec}
		m.RegisterType(element)
		array, _ := m.TypeForOID(entry.arrayOID)
		m.RegisterType(&pgtype.Type{Name: array.Name, OID: entry.arrayOID, Codec: &pgtype.ArrayCodec{ElementType: element}})
	}
	return m
}

// decodeText shares OID-aware value normalization between WAL and snapshot COPY.
func decodeText(m *pgtype.Map, oid uint32, data []byte) (any, error) {
	dt, ok := m.TypeForOID(oid)
	if !ok {
		return string(data), nil
	}
	value, err := dt.Codec.DecodeValue(m, oid, pgtype.TextFormatCode, data)
	if err != nil {
		return nil, err
	}
	if array, ok := dt.Codec.(*pgtype.ArrayCodec); ok {
		if elements, ok := value.([]any); ok {
			for i, element := range elements {
				elements[i] = normalizeValue(array.ElementType.OID, element)
			}
		}
		return value, nil
	}
	if uuid, ok := value.([16]byte); ok && oid == pgtype.UUIDOID {
		return uuidString(uuid), nil
	}
	return normalizeValue(oid, value), nil
}

// infiniteTime prints PostgreSQL infinity spelling but serializes as JSON/YAML null.
type infiniteTime pgtype.InfinityModifier

// String renders the value as "infinity" or "-infinity".
func (t infiniteTime) String() string { return pgtype.InfinityModifier(t).String() }

// MarshalJSON encodes the value as JSON null.
func (infiniteTime) MarshalJSON() ([]byte, error) { return []byte("null"), nil }

// MarshalYAML encodes the value as YAML null.
func (infiniteTime) MarshalYAML() (any, error) { return nil, nil }

// unmarshalJSONNumber keeps number lexemes and rejects trailing JSON values or garbage.
func unmarshalJSONNumber(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return errors.New("unexpected trailing JSON value")
	}
	return nil
}

// normalizeValue converts only the SQL type identified by oid, never nested JSON data.
func normalizeValue(oid uint32, value any) any {
	if modifier, ok := value.(pgtype.InfinityModifier); ok {
		return infiniteTime(modifier)
	}
	switch oid {
	case pgtype.TimestampOID:
		if timestamp, ok := value.(time.Time); ok {
			return timestamp.Format("2006-01-02T15:04:05.999999")
		}
	case pgtype.IntervalOID:
		if interval, ok := value.(pgtype.Interval); ok {
			return intervalMicroseconds(interval)
		}
	}
	return value
}

// intervalMicroseconds uses the average Gregorian month without integer overflow.
func intervalMicroseconds(interval pgtype.Interval) json.Number {
	var months, days, micros big.Int
	months.Mul(big.NewInt(int64(interval.Months)), big.NewInt(2629800000000))
	days.Mul(big.NewInt(int64(interval.Days)), big.NewInt(86400000000))
	micros.SetInt64(interval.Microseconds)
	micros.Add(&micros, &months)
	micros.Add(&micros, &days)
	return json.Number(micros.String())
}

// uuidString formats a raw UUID as its canonical lowercase hex string.
func uuidString(b [16]byte) string {
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}
