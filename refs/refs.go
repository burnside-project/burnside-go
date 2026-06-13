// Package refs defines the shared types for the refs.json contract between
// pg-cdc (producer) and pg-warehouse (consumer): the registry of named
// snapshots — mutable branches (main, staging) and immutable tags
// (raw@<rfc3339nano>) — per table, used by consumers for ref resolution and
// time travel, plus the structured contract diff that the /v1/diff endpoint
// and the promote gate return.
//
// This package is types-only. The producer's operational logic — creating
// tags, advancing branches, pruning, computing diffs, and reading/writing
// against a specific sink — lives in pg-cdc. Only the serializable shapes and
// io-based Read/Write are shared here so every repo agrees on the wire format.
//
// For Iceberg sinks SnapshotID is the real Iceberg snapshot id; for
// parquet_hive sinks SnapshotID is empty and consumers fall back to
// BaseEpoch + LatestDeltaEpoch.
package refs

import (
	"encoding/json"
	"fmt"
	"io"
)

// RefsPath is the conventional location of refs.json alongside manifest.json
// in the sink.
const RefsPath = "refs.json"

// Reserved branch names; tags use TagPrefix.
const (
	BranchMain    = "main"
	BranchStaging = "staging"
	TagPrefix     = "raw@"
)

// CurrentVersion is the on-disk schema version of refs.json. Bumping it is a
// breaking change; readers must reject newer files they don't understand.
const CurrentVersion = 1

// Kind distinguishes mutable branches from immutable tags.
type Kind string

const (
	KindBranch Kind = "branch"
	KindTag    Kind = "tag"
)

// ColumnRef is a tight snapshot of a column's identity at the moment a
// TableRef was committed — enough for a consumer to classify a column-level
// change as additive, removing, or in-place evolution. Storage-engine
// specifics (parquet type) are deliberately omitted; they derive from PgType.
type ColumnRef struct {
	Name      string `json:"name"`
	PgType    string `json:"pg_type"`
	Precision int    `json:"precision,omitempty"`
	Scale     int    `json:"scale,omitempty"`
	Nullable  bool   `json:"nullable,omitempty"`
}

// TableRef captures everything a consumer needs to read a specific table at a
// specific snapshot, regardless of storage format. PolicyVersion, ConfirmedLSN
// and Columns are populated when available and zero-value otherwise; older
// refs.json files predating a field decode cleanly.
type TableRef struct {
	SnapshotID        string      `json:"snapshot_id,omitempty"`
	BaseEpoch         int64       `json:"base_epoch"`
	LatestDeltaEpoch  int64       `json:"latest_delta_epoch"`
	SchemaVersion     int         `json:"schema_version"`
	PolicyVersion     int         `json:"policy_version,omitempty"`
	ManifestUpdatedAt string      `json:"manifest_updated_at"`
	ConfirmedLSN      string      `json:"confirmed_lsn,omitempty"`
	Columns           []ColumnRef `json:"columns,omitempty"`
}

// Ref is a named pointer at one TableRef per active table.
type Ref struct {
	Kind      Kind                `json:"kind"`
	CreatedAt string              `json:"created_at"`
	Tables    map[string]TableRef `json:"tables"`
}

// Refs is the root document persisted as refs.json. Producer-side concurrency
// control and mutation helpers are intentionally NOT part of this shared type.
type Refs struct {
	Version int            `json:"version"`
	Refs    map[string]Ref `json:"refs"`
}

// New returns an empty Refs at the current schema version.
func New() *Refs {
	return &Refs{Version: CurrentVersion, Refs: make(map[string]Ref)}
}

// Read parses refs.json from JSON. This is a pure decode — callers handle
// file-not-found (an empty Refs) at the storage layer. Files newer than
// CurrentVersion are rejected.
func Read(r io.Reader) (*Refs, error) {
	var rf Refs
	if err := json.NewDecoder(r).Decode(&rf); err != nil {
		return nil, fmt.Errorf("parse refs: %w", err)
	}
	if rf.Version > CurrentVersion {
		return nil, fmt.Errorf("refs.json version %d not supported (max %d)", rf.Version, CurrentVersion)
	}
	if rf.Refs == nil {
		rf.Refs = make(map[string]Ref)
	}
	return &rf, nil
}

// Write serializes refs.json to JSON (2-space indent, matching manifest).
func (rf *Refs) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rf)
}

// DiffVerdict classifies a contract diff as safe to promote or as requiring
// operator review. Surfaced in the diff payload and used by the promote gate.
type DiffVerdict string

const (
	VerdictNoChange     DiffVerdict = "no_change"
	VerdictSafe         DiffVerdict = "safe"
	VerdictSchemaChange DiffVerdict = "schema_change"
	VerdictBreaking     DiffVerdict = "breaking"
	VerdictPolicyChange DiffVerdict = "policy_change"
)

// TableStatus categorises an individual table's contribution to a diff.
type TableStatus string

const (
	StatusUnchanged    TableStatus = "unchanged"
	StatusAdditive     TableStatus = "additive"
	StatusSchemaChange TableStatus = "schema_change"
	StatusPolicyChange TableStatus = "policy_change"
	StatusAdded        TableStatus = "added"
	StatusRemoved      TableStatus = "removed"
)

// ColumnChange describes a column whose definition shifted between two refs
// (same Name on both sides). Adds/removes are surfaced by name via TableDiff.
type ColumnChange struct {
	Name string    `json:"name"`
	From ColumnRef `json:"from"`
	To   ColumnRef `json:"to"`
}

// TableDiff is the per-table delta between two refs.
type TableDiff struct {
	Name                 string         `json:"name"`
	Status               TableStatus    `json:"status"`
	SchemaVersionFrom    int            `json:"schema_version_from"`
	SchemaVersionTo      int            `json:"schema_version_to"`
	PolicyVersionFrom    int            `json:"policy_version_from"`
	PolicyVersionTo      int            `json:"policy_version_to"`
	BaseEpochFrom        int64          `json:"base_epoch_from"`
	BaseEpochTo          int64          `json:"base_epoch_to"`
	LatestDeltaEpochFrom int64          `json:"latest_delta_epoch_from"`
	LatestDeltaEpochTo   int64          `json:"latest_delta_epoch_to"`
	SnapshotIDFrom       string         `json:"snapshot_id_from,omitempty"`
	SnapshotIDTo         string         `json:"snapshot_id_to,omitempty"`
	ColumnsAdded         []string       `json:"columns_added,omitempty"`
	ColumnsRemoved       []string       `json:"columns_removed,omitempty"`
	ColumnsChanged       []ColumnChange `json:"columns_changed,omitempty"`
}

// ContractDiff is the structured payload returned by the diff endpoint and
// consumed by the promote gate.
type ContractDiff struct {
	From        string      `json:"from"`
	To          string      `json:"to"`
	Verdict     DiffVerdict `json:"verdict"`
	PromoteSafe bool        `json:"promote_safe"`
	Tables      []TableDiff `json:"tables"`
}
