// Package manifest defines the shared types for the manifest.json contract
// between pg-cdc (producer) and pg-warehouse (consumer).
package manifest

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Manifest is the root structure of manifest.json.
type Manifest struct {
	Version    int                `json:"version"`
	Source     Source             `json:"source"`
	Tables     map[string]Table   `json:"tables"`
	Schemas    map[string]Schema  `json:"schemas"`
	Roles      map[string]Role    `json:"roles,omitempty"`
	Compaction CompactionState    `json:"compaction,omitempty"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// Source describes the PostgreSQL origin.
type Source struct {
	Name            string    `json:"name"`
	PostgresVersion string    `json:"postgres_version"`
	InitLSN         string    `json:"init_lsn"`
	InitTimestamp   time.Time `json:"init_timestamp"`
}

// Table describes a single table in the manifest.
type Table struct {
	Status           string   `json:"status"`                      // "active" or "excluded"
	Tags             []string `json:"tags,omitempty"`
	Partition        string   `json:"partition,omitempty"`
	PrimaryKey       []string `json:"primary_key,omitempty"`
	SchemaVersion    int      `json:"schema_version,omitempty"`
	BaseEpoch        int64    `json:"base_epoch"`
	LatestDeltaEpoch int64    `json:"latest_delta_epoch"`
	RowCount         int64    `json:"row_count,omitempty"`
	Path             string   `json:"path"`
	Reason           string   `json:"reason,omitempty"`
}

// Schema describes the column layout for a table.
type Schema struct {
	Version int      `json:"version"`
	Columns []Column `json:"columns"`
}

// Column describes a single column with its Postgres and Parquet types.
type Column struct {
	Name        string `json:"name"`
	PgType      string `json:"pg_type"`
	ParquetType string `json:"parquet_type"`
	Precision   int    `json:"precision,omitempty"`
	Scale       int    `json:"scale,omitempty"`
	Nullable    bool   `json:"nullable"`
}

// Role defines table and column access for a PostgreSQL role.
type Role struct {
	Tables map[string]RoleAccess `json:"tables"`
}

// RoleAccess defines column-level access for a table.
// AllColumns is true when the role has access to all columns.
// When AllColumns is false, Columns lists the specific accessible columns.
type RoleAccess struct {
	AllColumns bool
	Columns    []string
}

// MarshalJSON writes RoleAccess as either "*" or ["col1","col2"].
func (ra RoleAccess) MarshalJSON() ([]byte, error) {
	if ra.AllColumns {
		return json.Marshal(struct {
			Columns string `json:"columns"`
		}{Columns: "*"})
	}
	return json.Marshal(struct {
		Columns []string `json:"columns"`
	}{Columns: ra.Columns})
}

// UnmarshalJSON reads RoleAccess from either {"columns":"*"} or {"columns":["col1"]}.
func (ra *RoleAccess) UnmarshalJSON(data []byte) error {
	var raw struct {
		Columns json.RawMessage `json:"columns"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.Columns) == 0 {
		ra.AllColumns = true
		return nil
	}
	// Try string first ("*")
	var s string
	if err := json.Unmarshal(raw.Columns, &s); err == nil {
		if s == "*" {
			ra.AllColumns = true
			return nil
		}
		return fmt.Errorf("unexpected columns value: %q", s)
	}
	// Try array
	var cols []string
	if err := json.Unmarshal(raw.Columns, &cols); err != nil {
		return fmt.Errorf("columns must be \"*\" or array of strings: %w", err)
	}
	ra.AllColumns = false
	ra.Columns = cols
	return nil
}

// CompactionState records the last compaction run.
type CompactionState struct {
	LastCompactedAt    time.Time `json:"last_compacted_at,omitempty"`
	TombstoneRetention string   `json:"tombstone_retention,omitempty"`
}

// Read parses a manifest from JSON.
func Read(r io.Reader) (*Manifest, error) {
	var m Manifest
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Version < 1 {
		return nil, fmt.Errorf("unsupported manifest version: %d", m.Version)
	}
	return &m, nil
}

// Write serializes the manifest to JSON.
func (m *Manifest) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

// ActiveTables returns only tables with status "active".
func (m *Manifest) ActiveTables() map[string]Table {
	result := make(map[string]Table, len(m.Tables))
	for name, t := range m.Tables {
		if t.Status == "active" {
			result[name] = t
		}
	}
	return result
}

// TablesForRoles computes the union of table access across multiple roles.
// Returns a map of table name → accessible columns (union of all granted roles).
func (m *Manifest) TablesForRoles(roles []string) map[string]RoleAccess {
	result := make(map[string]RoleAccess)

	for _, roleName := range roles {
		role, ok := m.Roles[roleName]
		if !ok {
			continue
		}
		for table, access := range role.Tables {
			existing, exists := result[table]
			if !exists {
				result[table] = access
				continue
			}
			// Union: if either grants all columns, result is all columns
			if existing.AllColumns || access.AllColumns {
				result[table] = RoleAccess{AllColumns: true}
				continue
			}
			// Union the column lists
			result[table] = RoleAccess{
				AllColumns: false,
				Columns:    unionStrings(existing.Columns, access.Columns),
			}
		}
	}
	return result
}

func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		seen[s] = true
	}
	result := make([]string, 0, len(seen))
	for s := range seen {
		result = append(result, s)
	}
	return result
}
