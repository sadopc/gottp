package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// PerfBaseline holds timing baselines for requests.
type PerfBaseline struct {
	Version   string                    `json:"version"`
	CreatedAt time.Time                 `json:"created_at"`
	Entries   map[string]PerfBaseEntry  `json:"entries"` // keyed by request name
}

// PerfBaseEntry holds the baseline timing for a single request.
type PerfBaseEntry struct {
	Name     string        `json:"name"`
	Method   string        `json:"method"`
	URL      string        `json:"url"`
	Duration time.Duration `json:"duration_ns"`
	DurHuman string        `json:"duration"` // for human readability
}

// PerfComparison holds a comparison between current and baseline timings.
type PerfComparison struct {
	Name         string        `json:"name"`
	Method       string        `json:"method"`
	Current      time.Duration `json:"current_ns"`
	Baseline     time.Duration `json:"baseline_ns"`
	Delta        time.Duration `json:"delta_ns"`
	DeltaPercent float64       `json:"delta_percent"`
	Regressed    bool          `json:"regressed"`
	IsNew        bool          `json:"is_new"`
}

// SavePerfBaseline writes results as a performance baseline file.
func SavePerfBaseline(path string, results []Result) error {
	baseline := PerfBaseline{
		Version:   "1",
		CreatedAt: time.Now(),
		Entries:   make(map[string]PerfBaseEntry),
	}

	for _, r := range results {
		if r.Error != nil {
			continue // skip errored requests
		}
		baseline.Entries[r.Name] = PerfBaseEntry{
			Name:     r.Name,
			Method:   r.Method,
			URL:      r.URL,
			Duration: r.Duration,
			DurHuman: formatDuration(r.Duration),
		}
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling baseline: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing baseline: %w", err)
	}

	return nil
}

// LoadPerfBaseline reads a performance baseline file.
func LoadPerfBaseline(path string) (*PerfBaseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading baseline: %w", err)
	}

	var baseline PerfBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("parsing baseline: %w", err)
	}

	return &baseline, nil
}

// ComparePerfBaseline compares results against a baseline.
// threshold is the percentage increase that counts as a regression (e.g. 20.0 = 20%).
func ComparePerfBaseline(results []Result, baseline *PerfBaseline, threshold float64) []PerfComparison {
	var comparisons []PerfComparison

	for _, r := range results {
		if r.Error != nil {
			continue
		}

		comp := PerfComparison{
			Name:    r.Name,
			Method:  r.Method,
			Current: r.Duration,
		}

		entry, ok := baseline.Entries[r.Name]
		if !ok {
			comp.IsNew = true
			comparisons = append(comparisons, comp)
			continue
		}

		comp.Baseline = entry.Duration
		comp.Delta = r.Duration - entry.Duration

		if entry.Duration > 0 {
			comp.DeltaPercent = float64(comp.Delta) / float64(entry.Duration) * 100
		}

		if comp.DeltaPercent > threshold {
			comp.Regressed = true
		}

		comparisons = append(comparisons, comp)
	}

	return comparisons
}

// HasRegressions returns true if any comparisons show regressions.
func HasRegressions(comparisons []PerfComparison) bool {
	for _, c := range comparisons {
		if c.Regressed {
			return true
		}
	}
	return false
}
