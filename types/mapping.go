// Package types provides Postgres-to-Parquet type mapping and CDC metadata constants.
package types

import "strings"

// CDC metadata columns added to Parquet files.
const (
	ColOp        = "__op"         // Operation: "I" (insert), "U" (update), "D" (delete)
	ColEpoch     = "__epoch"      // Epoch number (INT64)
	ColDeletedAt = "__deleted_at" // Tombstone timestamp in base files (TIMESTAMP_MICROS or NULL)
)

// Operation constants for delta Parquet files.
const (
	OpInsert = "I"
	OpUpdate = "U"
	OpDelete = "D"
)

// ParquetType returns the Parquet physical/logical type for a Postgres type.
func ParquetType(pgType string) string {
	switch normalize(pgType) {
	case "integer", "int", "int4":
		return "INT32"
	case "bigint", "int8":
		return "INT64"
	case "smallint", "int2":
		return "INT32"
	case "boolean", "bool":
		return "BOOLEAN"
	case "real", "float4":
		return "FLOAT"
	case "double precision", "float8":
		return "DOUBLE"
	case "numeric", "decimal":
		return "DECIMAL"
	case "text", "character varying", "varchar":
		return "UTF8"
	case "character", "char", "bpchar":
		return "UTF8"
	case "timestamp without time zone", "timestamp":
		return "TIMESTAMP_MICROS"
	case "timestamp with time zone", "timestamptz":
		return "TIMESTAMP_MICROS"
	case "date":
		return "DATE"
	case "time without time zone", "time":
		return "TIME_MICROS"
	case "uuid":
		return "UUID"
	case "jsonb", "json":
		return "JSON"
	case "bytea":
		return "BYTE_ARRAY"
	default:
		return "UTF8" // fallback: treat as string
	}
}

// DuckDBType returns the DuckDB type for a Postgres type.
// Used by pg-warehouse when creating tables in raw.duckdb.
func DuckDBType(pgType string) string {
	switch normalize(pgType) {
	case "integer", "int", "int4":
		return "INTEGER"
	case "bigint", "int8":
		return "BIGINT"
	case "smallint", "int2":
		return "SMALLINT"
	case "boolean", "bool":
		return "BOOLEAN"
	case "real", "float4":
		return "FLOAT"
	case "double precision", "float8":
		return "DOUBLE"
	case "numeric", "decimal":
		return "DOUBLE"
	case "text", "character varying", "varchar":
		return "VARCHAR"
	case "character", "char", "bpchar":
		return "VARCHAR"
	case "timestamp without time zone", "timestamp":
		return "TIMESTAMP"
	case "timestamp with time zone", "timestamptz":
		return "TIMESTAMPTZ"
	case "date":
		return "DATE"
	case "time without time zone", "time":
		return "TIME"
	case "uuid":
		return "UUID"
	case "jsonb", "json":
		return "JSON"
	case "bytea":
		return "BLOB"
	default:
		return "VARCHAR"
	}
}

func normalize(pgType string) string {
	t := strings.ToLower(strings.TrimSpace(pgType))
	// Strip precision/scale: "numeric(10,2)" → "numeric"
	if idx := strings.IndexByte(t, '('); idx >= 0 {
		t = t[:idx]
	}
	// Strip array brackets: "integer[]" → "integer"
	t = strings.TrimSuffix(t, "[]")
	return t
}
