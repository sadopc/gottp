package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadPerfBaseline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")

	results := []Result{
		{Name: "Get Users", Method: "GET", URL: "https://api.example.com/users", Duration: 150 * time.Millisecond},
		{Name: "Create User", Method: "POST", URL: "https://api.example.com/users", Duration: 300 * time.Millisecond},
		{Name: "Failed", Method: "GET", URL: "https://api.example.com/fail", Error: os.ErrNotExist},
	}

	if err := SavePerfBaseline(path, results); err != nil {
		t.Fatal(err)
	}

	baseline, err := LoadPerfBaseline(path)
	if err != nil {
		t.Fatal(err)
	}

	if baseline.Version != "1" {
		t.Errorf("expected version 1, got %s", baseline.Version)
	}

	if len(baseline.Entries) != 2 {
		t.Fatalf("expected 2 entries (skipping error), got %d", len(baseline.Entries))
	}

	entry := baseline.Entries["Get Users"]
	if entry.Duration != 150*time.Millisecond {
		t.Errorf("expected 150ms, got %v", entry.Duration)
	}

	if _, ok := baseline.Entries["Failed"]; ok {
		t.Error("errored request should not be in baseline")
	}
}

func TestComparePerfBaseline(t *testing.T) {
	baseline := &PerfBaseline{
		Entries: map[string]PerfBaseEntry{
			"Fast": {Name: "Fast", Method: "GET", Duration: 100 * time.Millisecond},
			"Slow": {Name: "Slow", Method: "GET", Duration: 500 * time.Millisecond},
		},
	}

	results := []Result{
		{Name: "Fast", Method: "GET", Duration: 150 * time.Millisecond}, // 50% slower -> regression at 20% threshold
		{Name: "Slow", Method: "GET", Duration: 400 * time.Millisecond}, // 20% faster -> improvement
		{Name: "New", Method: "POST", Duration: 200 * time.Millisecond}, // new request
	}

	comparisons := ComparePerfBaseline(results, baseline, 20.0)

	if len(comparisons) != 3 {
		t.Fatalf("expected 3 comparisons, got %d", len(comparisons))
	}

	// Fast should be regressed (50% > 20% threshold)
	if !comparisons[0].Regressed {
		t.Error("Fast should be regressed")
	}
	if comparisons[0].DeltaPercent != 50.0 {
		t.Errorf("expected 50%% delta, got %.1f%%", comparisons[0].DeltaPercent)
	}

	// Slow should not be regressed (it improved)
	if comparisons[1].Regressed {
		t.Error("Slow should not be regressed")
	}
	if comparisons[1].DeltaPercent != -20.0 {
		t.Errorf("expected -20%% delta, got %.1f%%", comparisons[1].DeltaPercent)
	}

	// New should be marked as new
	if !comparisons[2].IsNew {
		t.Error("New should be marked as new")
	}
}

func TestHasRegressions(t *testing.T) {
	noRegress := []PerfComparison{
		{Name: "A", Regressed: false},
		{Name: "B", Regressed: false},
	}
	if HasRegressions(noRegress) {
		t.Error("expected no regressions")
	}

	withRegress := []PerfComparison{
		{Name: "A", Regressed: false},
		{Name: "B", Regressed: true},
	}
	if !HasRegressions(withRegress) {
		t.Error("expected regressions")
	}
}

func TestPrintPerfComparison(t *testing.T) {
	comparisons := []PerfComparison{
		{Name: "Fast", Method: "GET", Current: 150 * time.Millisecond, Baseline: 100 * time.Millisecond, Delta: 50 * time.Millisecond, DeltaPercent: 50.0, Regressed: true},
		{Name: "Improved", Method: "GET", Current: 80 * time.Millisecond, Baseline: 100 * time.Millisecond, Delta: -20 * time.Millisecond, DeltaPercent: -20.0},
		{Name: "Stable", Method: "GET", Current: 102 * time.Millisecond, Baseline: 100 * time.Millisecond, Delta: 2 * time.Millisecond, DeltaPercent: 2.0},
		{Name: "New", Method: "POST", Current: 200 * time.Millisecond, IsNew: true},
	}

	var buf bytes.Buffer
	PrintPerfComparison(&buf, comparisons, 20.0)

	output := buf.String()
	if output == "" {
		t.Error("expected non-empty output")
	}

	// Verify key content is present
	tests := []string{
		"Performance Comparison",
		"regression",
		"improvement",
		"Fast",
		"New",
	}
	for _, s := range tests {
		if !bytes.Contains([]byte(output), []byte(s)) {
			t.Errorf("expected output to contain %q", s)
		}
	}
}

func TestLoadPerfBaseline_NonExistent(t *testing.T) {
	_, err := LoadPerfBaseline("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
