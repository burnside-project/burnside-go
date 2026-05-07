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
        "base_epoch": 42,
        "latest_delta_epoch": 47
      },
      "freshness": {
        "last_committed_ts": "2026-05-06T12:00:00Z",
        "last_committed_lsn": "0/16B3740",
        "lag_seconds": 12.5,
        "is_stale": false,
        "stale_threshold_seconds": 300.0
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
	if got.Version.LatestDeltaEpoch != 47 {
		t.Errorf("latest_delta_epoch: got %d, want 47", got.Version.LatestDeltaEpoch)
	}
	if got.Freshness.IsStale {
		t.Errorf("is_stale: got true, want false")
	}
	if got.Freshness.StaleThresholdSeconds != 300.0 {
		t.Errorf("stale_threshold_seconds: got %v, want 300.0", got.Freshness.StaleThresholdSeconds)
	}
	if got.Freshness.LagSeconds != 12.5 {
		t.Errorf("lag_seconds: got %v, want 12.5 (sub-second precision must round-trip)", got.Freshness.LagSeconds)
	}
	if got.Freshness.LastCommittedLSN != "0/16B3740" {
		t.Errorf("last_committed_lsn: got %q, want 0/16B3740", got.Freshness.LastCommittedLSN)
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

func TestGetSnapshotResponseRoundTrip_Iceberg(t *testing.T) {
	input := `{
  "format": "iceberg",
  "snapshot_id": "7d8a9c2e",
  "base_path": "s3://burnside-raw/public.orders/",
  "iceberg_table": "burnside_raw.public_orders",
  "metadata_location": "s3://burnside-raw/public.orders/metadata/00002-uuid.metadata.json",
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
	if resp.Format != "iceberg" {
		t.Errorf("format: got %q, want iceberg", resp.Format)
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
	if resp.MetadataLocation == "" {
		t.Errorf("metadata_location should be set for iceberg format")
	}
	if resp.Version.SchemaVersion != 3 {
		t.Errorf("version.schema_version: got %d, want 3", resp.Version.SchemaVersion)
	}
}

func TestGetSnapshotResponseRoundTrip_ParquetHive(t *testing.T) {
	// Producers without Iceberg support: format=parquet_hive, no iceberg_table /
	// metadata_location, consumers read base + deltas from BasePath.
	input := `{
  "format": "parquet_hive",
  "base_path": "s3://burnside-raw/public.orders/",
  "version": {
    "schema_version": 3,
    "policy_version": 1,
    "base_epoch": 42,
    "latest_delta_epoch": 47
  }
}`

	var resp GetSnapshotResponse
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Format != "parquet_hive" {
		t.Errorf("format: got %q, want parquet_hive", resp.Format)
	}
	if resp.IcebergTable != "" || resp.MetadataLocation != "" {
		t.Errorf("parquet_hive should have empty iceberg fields, got %+v", resp)
	}
	if resp.Version.LatestDeltaEpoch != 47 {
		t.Errorf("latest_delta_epoch: got %d, want 47", resp.Version.LatestDeltaEpoch)
	}
}

func TestFreshnessStaleVerdict(t *testing.T) {
	f := Freshness{
		LastCommittedTS:       time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		LastCommittedLSN:      "0/16B3740",
		LagSeconds:            900.5,
		IsStale:               true,
		StaleThresholdSeconds: 300.0,
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
	if got.LagSeconds != 900.5 {
		t.Errorf("lag_seconds: got %v, want 900.5", got.LagSeconds)
	}
	if got.LastCommittedLSN != "0/16B3740" {
		t.Errorf("last_committed_lsn lost in round-trip: %q", got.LastCommittedLSN)
	}
}

// TestFreshnessOmitsOptional confirms that producers that don't populate
// LastCommittedLSN don't emit a stray empty field.
func TestFreshnessOmitsOptional(t *testing.T) {
	f := Freshness{
		LastCommittedTS:       time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		LagSeconds:            5,
		IsStale:               false,
		StaleThresholdSeconds: 300,
	}
	out, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, present := raw["last_committed_lsn"]; present {
		t.Errorf("last_committed_lsn should be omitted when empty: %s", string(out))
	}
}
