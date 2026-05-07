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
//
// v0.4.0 reconciliation note: the v0.3.0 shape was a clean-room design
// against pg-warehouse's needs. v0.4.0 absorbs operational fields pg-cdc
// already exposes (LastCommittedLSN, LatestDeltaEpoch, MetadataLocation,
// Format) and aligns numeric types with what pg-cdc already serializes
// (LagSeconds + StaleThresholdSeconds as float64, not int64). These are
// the only breaking type changes; the additive fields are all omitempty.
package contract

import "time"

// TableVersion is the monotonic stamp tuple a consumer diffs against last-run
// state to decide whether a table needs refresh.
//
// PolicyVersion is opaque to the consumer — it bumps whenever role grants or
// column-level access rules change on the producer, so a refresh re-reads
// even when schema and data are unchanged.
//
// LatestDeltaEpoch is meaningful for the legacy parquet_hive read path
// (consumer reads base + deltas above LatestDeltaEpoch). Iceberg consumers
// can ignore it. Optional / omitempty so producers that only serve Iceberg
// don't have to populate it.
type TableVersion struct {
	SchemaVersion    int   `json:"schema_version"`
	PolicyVersion    int   `json:"policy_version"`
	BaseEpoch        int64 `json:"base_epoch"`
	LatestDeltaEpoch int64 `json:"latest_delta_epoch,omitempty"`
}

// Freshness reports how recent a table's data is on the producer.
//
// IsStale is the producer's verdict, computed from StaleThresholdSeconds.
// Consumers should refuse to derive from stale raw without an explicit
// override (the --allow-stale flag in pg-warehouse).
//
// LastCommittedLSN is an optional Postgres-specific signal — useful for
// diagnostics when the producer is a Postgres CDC pipeline.
//
// LagSeconds and StaleThresholdSeconds are float64 so producers that
// measure with sub-second precision don't lose it at the API boundary.
type Freshness struct {
	LastCommittedTS       time.Time `json:"last_committed_ts"`
	LastCommittedLSN      string    `json:"last_committed_lsn,omitempty"`
	LagSeconds            float64   `json:"lag_seconds"`
	IsStale               bool      `json:"is_stale"`
	StaleThresholdSeconds float64   `json:"stale_threshold_seconds"`
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
//
// Format describes how the consumer should read the data:
//
//   - "iceberg" — IcebergTable + MetadataLocation are set; the consumer
//     uses an iceberg-go client (or DuckDB's iceberg_scan) on
//     MetadataLocation, optionally cross-checking SnapshotID.
//   - "parquet_hive" — IcebergTable / MetadataLocation are empty; the
//     consumer reads base + delta Parquet from BasePath using
//     Version.BaseEpoch and Version.LatestDeltaEpoch.
//
// Format may be empty when the producer hasn't been updated to set it;
// consumers should infer the format from the presence of MetadataLocation.
//
// SnapshotID is a string (Iceberg's int64 ID is serialized as a decimal
// string) so it can also represent non-Iceberg snapshot identifiers
// without a type change.
type GetSnapshotResponse struct {
	Format           string       `json:"format,omitempty"`
	SnapshotID       string       `json:"snapshot_id,omitempty"`
	BasePath         string       `json:"base_path"`
	IcebergTable     string       `json:"iceberg_table,omitempty"`
	MetadataLocation string       `json:"metadata_location,omitempty"`
	Version          TableVersion `json:"version"`
}
