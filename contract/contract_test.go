package contract

import (
	"encoding/json"
	"testing"
	"time"
)

func TestListTablesResponseRoundTrip(t *testing.T) {
	input := `{
  "tables": {
    "public.orders": {
      "version": {
        "schema_version": 3,
        "policy_version": 1,
        "base_epoch": 42
      },
      "freshness": {
        "last_committed_ts": "2026-05-06T12:00:00Z",
        "lag_seconds": 12,
        "is_stale": false,
        "stale_threshold_seconds": 300
      }
    }
  }
}`

	var resp ListTablesResponse
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	got, ok := resp.Tables["public.orders"]
	if !ok {
		t.Fatalf("missing public.orders")
	}
	if got.Version.SchemaVersion != 3 {
		t.Errorf("schema_version: got %d, want 3", got.Version.SchemaVersion)
	}
	if got.Version.PolicyVersion != 1 {
		t.Errorf("policy_version: got %d, want 1", got.Version.PolicyVersion)
	}
	if got.Version.BaseEpoch != 42 {
		t.Errorf("base_epoch: got %d, want 42", got.Version.BaseEpoch)
	}
	if got.Freshness.IsStale {
		t.Errorf("is_stale: got true, want false")
	}
	if got.Freshness.StaleThresholdSeconds != 300 {
		t.Errorf("stale_threshold_seconds: got %d, want 300", got.Freshness.StaleThresholdSeconds)
	}

	// Re-marshal and unmarshal to verify stability
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var resp2 ListTablesResponse
	if err := json.Unmarshal(out, &resp2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if resp2.Tables["public.orders"].Version != got.Version {
		t.Errorf("version drift after round-trip")
	}
	if !resp2.Tables["public.orders"].Freshness.LastCommittedTS.Equal(got.Freshness.LastCommittedTS) {
		t.Errorf("timestamp drift after round-trip")
	}
}

func TestGetSnapshotResponseRoundTrip(t *testing.T) {
	input := `{
  "snapshot_id": "7d8a9c2e",
  "base_path": "s3://burnside-raw/public.orders/",
  "iceberg_table": "burnside_raw.public_orders",
  "version": {
    "schema_version": 3,
    "policy_version": 1,
    "base_epoch": 42
  }
}`

	var resp GetSnapshotResponse
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.SnapshotID != "7d8a9c2e" {
		t.Errorf("snapshot_id: got %q, want 7d8a9c2e", resp.SnapshotID)
	}
	if resp.BasePath != "s3://burnside-raw/public.orders/" {
		t.Errorf("base_path mismatch: %q", resp.BasePath)
	}
	if resp.IcebergTable != "burnside_raw.public_orders" {
		t.Errorf("iceberg_table mismatch: %q", resp.IcebergTable)
	}
	if resp.Version.SchemaVersion != 3 {
		t.Errorf("version.schema_version: got %d, want 3", resp.Version.SchemaVersion)
	}
}

func TestFreshnessStaleVerdict(t *testing.T) {
	f := Freshness{
		LastCommittedTS:       time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		LagSeconds:            900,
		IsStale:               true,
		StaleThresholdSeconds: 300,
	}
	out, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Freshness
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.IsStale {
		t.Errorf("is_stale lost in round-trip")
	}
	if got.LagSeconds != 900 {
		t.Errorf("lag_seconds: got %d, want 900", got.LagSeconds)
	}
}
