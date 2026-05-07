// Package contract defines the wire-level types for the cross-repo API
// between pg-cdc (producer) and pg-warehouse (consumer).
//
// These are distinct from manifest types: manifest describes the on-disk
// file format, contract describes the HTTP/JSON responses pg-cdc serves to
// pg-warehouse during --refresh. Both are semver-stable; breaking changes
// require coordinated PRs across both repos.
//
// Phase 1 endpoints:
//   - GET /v1/tables                     → ListTablesResponse
//   - GET /v1/tables/{name}/snapshot     → GetSnapshotResponse
//   - GET /v1/tables/{name}/freshness    → Freshness
package contract

import "time"

// TableVersion is the monotonic stamp tuple a consumer diffs against last-run
// state to decide whether a table needs refresh.
//
// PolicyVersion is opaque to the consumer — it bumps whenever role grants or
// column-level access rules change on the producer, so a refresh re-reads
// even when schema and data are unchanged.
type TableVersion struct {
	SchemaVersion int   `json:"schema_version"`
	PolicyVersion int   `json:"policy_version"`
	BaseEpoch     int64 `json:"base_epoch"`
}

// Freshness reports how recent a table's data is on the producer.
//
// IsStale is the producer's verdict, computed from StaleThresholdSeconds.
// Consumers should refuse to derive from stale raw without an explicit
// override (the --allow-stale flag in pg-warehouse).
type Freshness struct {
	LastCommittedTS      time.Time `json:"last_committed_ts"`
	LagSeconds           int64     `json:"lag_seconds"`
	IsStale              bool      `json:"is_stale"`
	StaleThresholdSeconds int64    `json:"stale_threshold_seconds"`
}

// ListTablesResponse is the body of GET /v1/tables.
//
// Keyed by fully-qualified table name (e.g., "public.orders").
type ListTablesResponse struct {
	Tables map[string]TableEntry `json:"tables"`
}

// TableEntry pairs version stamps with freshness so a single list_tables
// call answers both "did anything change?" and "is it stale?".
type TableEntry struct {
	Version   TableVersion `json:"version"`
	Freshness Freshness    `json:"freshness"`
}

// GetSnapshotResponse is the body of GET /v1/tables/{name}/snapshot.
//
// The consumer reads Parquet directly from BasePath using its own AWS SSO
// profile — pg-cdc does not proxy data. Lake Formation gates at the
// Glue/S3 layer.
type GetSnapshotResponse struct {
	SnapshotID   string       `json:"snapshot_id"`
	BasePath     string       `json:"base_path"`
	IcebergTable string       `json:"iceberg_table"`
	Version      TableVersion `json:"version"`
}
