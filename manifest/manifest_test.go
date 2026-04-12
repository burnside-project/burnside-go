package manifest

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestReadWrite(t *testing.T) {
	input := `{
  "version": 1,
  "source": {
    "name": "prod-pg",
    "postgres_version": "16.2",
    "init_lsn": "0/16B3740",
    "init_timestamp": "2026-04-12T08:00:00Z"
  },
  "tables": {
    "public.orders": {
      "status": "active",
      "tags": [],
      "primary_key": ["id"],
      "schema_version": 1,
      "base_epoch": 0,
      "latest_delta_epoch": 2,
      "row_count": 100,
      "path": "public.orders"
    },
    "public.tbl_sessions": {
      "status": "excluded",
      "tags": ["ephemeral"],
      "path": "",
      "base_epoch": 0,
      "latest_delta_epoch": 0,
      "reason": "policy:ephemeral"
    }
  },
  "schemas": {
    "public.orders": {
      "version": 1,
      "columns": [
        {"name": "id", "pg_type": "bigint", "parquet_type": "INT64", "nullable": false},
        {"name": "total", "pg_type": "numeric(10,2)", "parquet_type": "DECIMAL", "precision": 10, "scale": 2, "nullable": false}
      ]
    }
  },
  "roles": {
    "analyst": {
      "tables": {
        "public.orders": {"columns": "*"},
        "public.products": {"columns": ["id", "name", "price"]}
      }
    }
  },
  "updated_at": "2026-04-12T14:00:00Z"
}`

	m, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
	if m.Source.InitLSN != "0/16B3740" {
		t.Errorf("InitLSN = %q", m.Source.InitLSN)
	}
	if len(m.Tables) != 2 {
		t.Errorf("Tables count = %d, want 2", len(m.Tables))
	}

	orders := m.Tables["public.orders"]
	if orders.Status != "active" {
		t.Errorf("orders.Status = %q", orders.Status)
	}
	if orders.LatestDeltaEpoch != 2 {
		t.Errorf("orders.LatestDeltaEpoch = %d", orders.LatestDeltaEpoch)
	}

	sessions := m.Tables["public.tbl_sessions"]
	if sessions.Status != "excluded" {
		t.Errorf("sessions.Status = %q", sessions.Status)
	}
	if sessions.Reason != "policy:ephemeral" {
		t.Errorf("sessions.Reason = %q", sessions.Reason)
	}

	schema := m.Schemas["public.orders"]
	if len(schema.Columns) != 2 {
		t.Errorf("schema columns = %d", len(schema.Columns))
	}
	if schema.Columns[1].Precision != 10 {
		t.Errorf("precision = %d", schema.Columns[1].Precision)
	}

	// Role access
	analyst := m.Roles["analyst"]
	if !analyst.Tables["public.orders"].AllColumns {
		t.Error("analyst.orders should have AllColumns")
	}
	products := analyst.Tables["public.products"]
	if products.AllColumns {
		t.Error("analyst.products should NOT have AllColumns")
	}
	if len(products.Columns) != 3 {
		t.Errorf("analyst.products columns = %d", len(products.Columns))
	}

	// Round-trip write
	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	m2, err := Read(&buf)
	if err != nil {
		t.Fatalf("re-Read: %v", err)
	}
	if m2.Source.InitLSN != m.Source.InitLSN {
		t.Error("round-trip failed")
	}
}

func TestActiveTables(t *testing.T) {
	m := &Manifest{
		Tables: map[string]Table{
			"a": {Status: "active"},
			"b": {Status: "excluded"},
			"c": {Status: "active"},
		},
	}
	active := m.ActiveTables()
	if len(active) != 2 {
		t.Errorf("ActiveTables = %d, want 2", len(active))
	}
	if _, ok := active["b"]; ok {
		t.Error("excluded table should not be in ActiveTables")
	}
}

func TestTablesForRoles_Union(t *testing.T) {
	m := &Manifest{
		Roles: map[string]Role{
			"base": {
				Tables: map[string]RoleAccess{
					"orders":   {AllColumns: true},
					"payments": {Columns: []string{"id", "amount"}},
				},
			},
			"pii": {
				Tables: map[string]RoleAccess{
					"payments": {Columns: []string{"id", "card_number"}},
					"users":    {AllColumns: true},
				},
			},
		},
	}

	access := m.TablesForRoles([]string{"base", "pii"})

	// orders: only from base, all columns
	if !access["orders"].AllColumns {
		t.Error("orders should have AllColumns")
	}

	// payments: union of [id, amount] + [id, card_number] = [id, amount, card_number]
	payments := access["payments"]
	if payments.AllColumns {
		t.Error("payments should not have AllColumns")
	}
	if len(payments.Columns) != 3 {
		t.Errorf("payments columns = %v, want 3", payments.Columns)
	}

	// users: only from pii
	if !access["users"].AllColumns {
		t.Error("users should have AllColumns")
	}
}

func TestTablesForRoles_UnknownRole(t *testing.T) {
	m := &Manifest{
		Roles: map[string]Role{
			"analyst": {Tables: map[string]RoleAccess{"orders": {AllColumns: true}}},
		},
	}
	access := m.TablesForRoles([]string{"nonexistent"})
	if len(access) != 0 {
		t.Errorf("unknown role should return empty, got %d", len(access))
	}
}

func TestReadInvalidVersion(t *testing.T) {
	_, err := Read(strings.NewReader(`{"version": 0}`))
	if err == nil {
		t.Error("expected error for version 0")
	}
}

func TestReadInvalidJSON(t *testing.T) {
	_, err := Read(strings.NewReader(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestManifestWriteTimestamp(t *testing.T) {
	m := &Manifest{
		Version:   1,
		UpdatedAt: time.Date(2026, 4, 12, 14, 0, 0, 0, time.UTC),
		Tables:    map[string]Table{},
		Schemas:   map[string]Schema{},
	}
	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "2026-04-12T14:00:00Z") {
		t.Errorf("timestamp not in output: %s", buf.String())
	}
}
