package types

import "testing"

func TestParquetType(t *testing.T) {
	tests := []struct {
		pgType string
		want   string
	}{
		{"integer", "INT32"},
		{"bigint", "INT64"},
		{"INT8", "INT64"},
		{"boolean", "BOOLEAN"},
		{"text", "UTF8"},
		{"character varying", "UTF8"},
		{"VARCHAR", "UTF8"},
		{"numeric(10,2)", "DECIMAL"},
		{"NUMERIC", "DECIMAL"},
		{"timestamptz", "TIMESTAMP_MICROS"},
		{"timestamp with time zone", "TIMESTAMP_MICROS"},
		{"timestamp without time zone", "TIMESTAMP_MICROS"},
		{"date", "DATE"},
		{"uuid", "UUID"},
		{"jsonb", "JSON"},
		{"json", "JSON"},
		{"bytea", "BYTE_ARRAY"},
		{"double precision", "DOUBLE"},
		{"real", "FLOAT"},
		{"unknown_type", "UTF8"},
		{"integer[]", "INT32"},
	}

	for _, tt := range tests {
		got := ParquetType(tt.pgType)
		if got != tt.want {
			t.Errorf("ParquetType(%q) = %q, want %q", tt.pgType, got, tt.want)
		}
	}
}

func TestDuckDBType(t *testing.T) {
	tests := []struct {
		pgType string
		want   string
	}{
		{"integer", "INTEGER"},
		{"bigint", "BIGINT"},
		{"text", "VARCHAR"},
		{"numeric(10,2)", "DOUBLE"},
		{"timestamptz", "TIMESTAMPTZ"},
		{"uuid", "UUID"},
		{"jsonb", "JSON"},
		{"bytea", "BLOB"},
		{"unknown", "VARCHAR"},
	}

	for _, tt := range tests {
		got := DuckDBType(tt.pgType)
		if got != tt.want {
			t.Errorf("DuckDBType(%q) = %q, want %q", tt.pgType, got, tt.want)
		}
	}
}
