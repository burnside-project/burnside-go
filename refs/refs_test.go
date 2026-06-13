package refs

import (
	"bytes"
	"strings"
	"testing"
)

func TestRefsRoundTrip(t *testing.T) {
	in := New()
	in.Refs[BranchMain] = Ref{
		Kind:      KindBranch,
		CreatedAt: "2026-01-01T00:00:00Z",
		Tables: map[string]TableRef{
			"public.users": {
				BaseEpoch:         1,
				LatestDeltaEpoch:  2,
				SchemaVersion:     3,
				ManifestUpdatedAt: "2026-01-01T00:00:01Z",
				Columns:           []ColumnRef{{Name: "id", PgType: "int8"}},
			},
		},
	}
	var buf bytes.Buffer
	if err := in.Write(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := Read(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out.Version != CurrentVersion {
		t.Errorf("version = %d, want %d", out.Version, CurrentVersion)
	}
	tr := out.Refs[BranchMain].Tables["public.users"]
	if tr.BaseEpoch != 1 || tr.SchemaVersion != 3 || len(tr.Columns) != 1 || tr.Columns[0].Name != "id" {
		t.Errorf("round-trip lost data: %+v", tr)
	}
}

func TestReadRejectsNewerVersion(t *testing.T) {
	if _, err := Read(strings.NewReader(`{"version":999,"refs":{}}`)); err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestReadNilRefsMapInitialized(t *testing.T) {
	rf, err := Read(strings.NewReader(`{"version":1}`))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if rf.Refs == nil {
		t.Fatal("Refs map should be initialized, not nil")
	}
}
