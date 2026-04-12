package epoch

import (
	"testing"
)

func TestParseFromFilename(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		err   bool
	}{
		{"epoch=000042.parquet", 42, false},
		{"epoch=000001.parquet", 1, false},
		{"epoch=000000.parquet", 0, false},
		{"epoch=999999.parquet", 999999, false},
		{"path/to/epoch=000005.parquet", 5, false},
		{"base_000000.parquet", 0, true},
		{"random.txt", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseFromFilename(tt.input)
		if tt.err && err == nil {
			t.Errorf("ParseFromFilename(%q) expected error", tt.input)
			continue
		}
		if !tt.err && err != nil {
			t.Errorf("ParseFromFilename(%q) = error %v", tt.input, err)
			continue
		}
		if !tt.err && got != tt.want {
			t.Errorf("ParseFromFilename(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFormatFilename(t *testing.T) {
	if got := FormatFilename(42); got != "epoch=000042.parquet" {
		t.Errorf("FormatFilename(42) = %q", got)
	}
	if got := FormatFilename(0); got != "epoch=000000.parquet" {
		t.Errorf("FormatFilename(0) = %q", got)
	}
}

func TestFilterAfter(t *testing.T) {
	files := []string{
		"public.orders/deltas/epoch=000001.parquet",
		"public.orders/deltas/epoch=000003.parquet",
		"public.orders/deltas/epoch=000002.parquet",
		"public.orders/deltas/epoch=000005.parquet",
		"public.orders/deltas/epoch=000004.parquet",
	}

	result, err := FilterAfter(files, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("FilterAfter(2) = %d files, want 3", len(result))
	}
	// Should be sorted ascending
	for i, want := range []string{
		"public.orders/deltas/epoch=000003.parquet",
		"public.orders/deltas/epoch=000004.parquet",
		"public.orders/deltas/epoch=000005.parquet",
	} {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}

func TestFilterAfter_Empty(t *testing.T) {
	files := []string{
		"epoch=000001.parquet",
		"epoch=000002.parquet",
	}
	result, _ := FilterAfter(files, 5)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestEpoch_After(t *testing.T) {
	a := Epoch{Number: 5}
	b := Epoch{Number: 3}
	if !a.After(b) {
		t.Error("5 should be after 3")
	}
	if b.After(a) {
		t.Error("3 should not be after 5")
	}
}
