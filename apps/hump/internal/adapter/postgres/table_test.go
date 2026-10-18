package postgres

import "testing"

func TestParseTableType(t *testing.T) {
	tests := []struct {
		literal string
		want    tableType
	}{
		{literal: "regular", want: tableTypeRegular},
		{literal: "hypertable", want: tableTypeHypertable},
		{literal: "partitioned", want: tableTypePartitioned},
		{literal: "", want: tableTypeRegular},
		{literal: "Hypertable", want: tableTypeRegular},
		{literal: "distributed_hypertable", want: tableTypeRegular},
	}

	for _, tt := range tests {
		t.Run(tt.literal, func(t *testing.T) {
			if got := parseTableType(tt.literal); got != tt.want {
				t.Errorf("parseTableType(%q) = %v, want %v", tt.literal, got, tt.want)
			}
		})
	}
}

func TestParseTableTypeRoundTripsQueryLiterals(t *testing.T) {
	for _, want := range []tableType{tableTypeRegular, tableTypeHypertable, tableTypePartitioned} {
		literal := tableTypeLiteral(want)
		if got := parseTableType(literal); got != want {
			t.Errorf("parseTableType(%q) = %v, want %v", literal, got, want)
		}
	}
}
