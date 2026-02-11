package diff

import (
	"testing"
)

func TestDiffLinesSame(t *testing.T) {
	result := DiffLines("a\nb\nc", "a\nb\nc")
	for _, line := range result {
		if line.Type != Same {
			t.Errorf("expected all Same, got %v for %q", line.Type, line.Content)
		}
	}
	if len(result) != 3 {
		t.Errorf("expected 3 lines, got %d", len(result))
	}
}

func TestDiffLinesAdded(t *testing.T) {
	result := DiffLines("a\nc", "a\nb\nc")
	added := 0
	for _, line := range result {
		if line.Type == Added {
			added++
			if line.Content != "b" {
				t.Errorf("expected added line 'b', got %q", line.Content)
			}
		}
	}
	if added != 1 {
		t.Errorf("expected 1 added line, got %d", added)
	}
}

func TestDiffLinesRemoved(t *testing.T) {
	result := DiffLines("a\nb\nc", "a\nc")
	removed := 0
	for _, line := range result {
		if line.Type == Removed {
			removed++
			if line.Content != "b" {
				t.Errorf("expected removed line 'b', got %q", line.Content)
			}
		}
	}
	if removed != 1 {
		t.Errorf("expected 1 removed line, got %d", removed)
	}
}

func TestDiffLinesEmpty(t *testing.T) {
	result := DiffLines("", "")
	if len(result) != 0 {
		t.Errorf("expected 0 lines, got %d", len(result))
	}
}

func TestDiffLinesAllNew(t *testing.T) {
	result := DiffLines("", "a\nb\nc")
	if len(result) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(result))
	}
	for _, line := range result {
		if line.Type != Added {
			t.Errorf("expected Added, got %v", line.Type)
		}
	}
}

func TestDiffLinesAllRemoved(t *testing.T) {
	result := DiffLines("a\nb\nc", "")
	if len(result) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(result))
	}
	for _, line := range result {
		if line.Type != Removed {
			t.Errorf("expected Removed, got %v", line.Type)
		}
	}
}

func TestDiffLinesComplex(t *testing.T) {
	a := "line1\nline2\nline3\nline4\nline5"
	b := "line1\nline2modified\nline3\nline4.5\nline5"
	result := DiffLines(a, b)

	// Should have removals for line2 and line4, additions for line2modified and line4.5
	hasRemoved := false
	hasAdded := false
	for _, line := range result {
		if line.Type == Removed {
			hasRemoved = true
		}
		if line.Type == Added {
			hasAdded = true
		}
	}
	if !hasRemoved || !hasAdded {
		t.Error("expected both additions and removals in complex diff")
	}
}

func TestDiffLinesLineNumbers(t *testing.T) {
	result := DiffLines("a\nb", "a\nc")
	for _, line := range result {
		if line.Type == Same && line.OldLine <= 0 {
			t.Errorf("same lines should have positive OldLine, got %d", line.OldLine)
		}
		if line.Type == Removed && line.NewLine != -1 {
			t.Errorf("removed lines should have NewLine=-1, got %d", line.NewLine)
		}
		if line.Type == Added && line.OldLine != -1 {
			t.Errorf("added lines should have OldLine=-1, got %d", line.OldLine)
		}
	}
}
