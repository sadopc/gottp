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

// --- Word-level diff tests ---

func TestDiffWords_Identical(t *testing.T) {
	result := DiffWords("hello world foo", "hello world foo")
	for _, w := range result {
		if w.Type != Same {
			t.Errorf("expected all Same, got %v for %q", w.Type, w.Content)
		}
	}
	// Reconstruct the string from Same tokens.
	var reconstructed string
	for _, w := range result {
		reconstructed += w.Content
	}
	if reconstructed != "hello world foo" {
		t.Errorf("reconstructed %q does not match original", reconstructed)
	}
}

func TestDiffWords_CompletelyDifferent(t *testing.T) {
	result := DiffWords("alpha beta", "gamma delta")
	hasRemoved := false
	hasAdded := false
	for _, w := range result {
		if w.Type == Removed {
			hasRemoved = true
		}
		if w.Type == Added {
			hasAdded = true
		}
	}
	if !hasRemoved {
		t.Error("expected removed words")
	}
	if !hasAdded {
		t.Error("expected added words")
	}

	// The Same tokens should only be whitespace (if any).
	for _, w := range result {
		if w.Type == Same {
			for _, r := range w.Content {
				if r != ' ' && r != '\t' {
					t.Errorf("expected only whitespace in Same tokens, got %q", w.Content)
					break
				}
			}
		}
	}
}

func TestDiffWords_SingleWordChange(t *testing.T) {
	result := DiffWords("hello world", "hello earth")
	// "hello" and the space should be Same, "world" removed, "earth" added.
	var removed, added []string
	for _, w := range result {
		switch w.Type {
		case Removed:
			removed = append(removed, w.Content)
		case Added:
			added = append(added, w.Content)
		case Same:
			if w.Content == "hello" {
				// expected
			}
		}
	}
	if len(removed) != 1 || removed[0] != "world" {
		t.Errorf("expected removed=[world], got %v", removed)
	}
	if len(added) != 1 || added[0] != "earth" {
		t.Errorf("expected added=[earth], got %v", added)
	}
}

func TestDiffWords_WordAddedRemoved(t *testing.T) {
	// Word removed: "a b c" -> "a c"
	result := DiffWords("a b c", "a c")
	removedCount := 0
	for _, w := range result {
		if w.Type == Removed {
			removedCount++
		}
	}
	if removedCount == 0 {
		t.Error("expected at least one removed word for 'a b c' -> 'a c'")
	}

	// Word added: "a b c" -> "a x b c"
	result2 := DiffWords("a b c", "a x b c")
	addedCount := 0
	for _, w := range result2 {
		if w.Type == Added {
			addedCount++
		}
	}
	if addedCount == 0 {
		t.Error("expected at least one added word for 'a b c' -> 'a x b c'")
	}
}

func TestDiffWords_EmptyInputs(t *testing.T) {
	// Both empty
	result := DiffWords("", "")
	if len(result) != 0 {
		t.Errorf("expected 0 word diffs for empty inputs, got %d", len(result))
	}

	// One empty, one non-empty
	result2 := DiffWords("", "hello world")
	allAdded := true
	for _, w := range result2 {
		if w.Type != Added {
			allAdded = false
		}
	}
	if !allAdded {
		t.Error("expected all Added when first input is empty")
	}

	result3 := DiffWords("hello world", "")
	allRemoved := true
	for _, w := range result3 {
		if w.Type != Removed {
			allRemoved = false
		}
	}
	if !allRemoved {
		t.Error("expected all Removed when second input is empty")
	}
}

// --- DiffLinesWithWords tests ---

func TestDiffLinesWithWords_ModifiedLines(t *testing.T) {
	a := "line one here\nline two here"
	b := "line one changed\nline two here"
	result := DiffLinesWithWords(a, b)

	// The first line is modified (removed "here", added "changed").
	// We should have: Removed(line one here), Added(line one changed), Same(line two here).
	foundModifiedWithWords := false
	for _, r := range result {
		if r.Words != nil {
			foundModifiedWithWords = true
			// Verify word-level detail exists and has mixed types.
			hasSame := false
			hasChange := false
			for _, w := range r.Words {
				if w.Type == Same {
					hasSame = true
				}
				if w.Type == Added || w.Type == Removed {
					hasChange = true
				}
			}
			if !hasSame || !hasChange {
				t.Error("expected both Same and changed words in word-level diff")
			}
		}
	}
	if !foundModifiedWithWords {
		t.Error("expected at least one line with word-level detail")
	}
}

func TestDiffLinesWithWords_PureAddsRemoves(t *testing.T) {
	// Pure addition: lines added at the end have no matching removed counterpart.
	a := "line one"
	b := "line one\nline two\nline three"
	result := DiffLinesWithWords(a, b)

	for _, r := range result {
		if r.Type == Added && r.Words != nil {
			t.Errorf("pure Added line should have nil Words, got %v", r.Words)
		}
	}

	// Pure removal: lines removed with no matching added counterpart.
	a2 := "line one\nline two\nline three"
	b2 := "line one"
	result2 := DiffLinesWithWords(a2, b2)

	for _, r := range result2 {
		if r.Type == Removed && r.Words != nil {
			t.Errorf("pure Removed line should have nil Words, got %v", r.Words)
		}
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world  foo", []string{"hello", " ", "world", "  ", "foo"}},
		{"", nil},
		{"  leading", []string{"  ", "leading"}},
		{"trailing  ", []string{"trailing", "  "}},
		{"single", []string{"single"}},
		{"  ", []string{"  "}},
	}

	for _, tc := range tests {
		got := splitWords(tc.input)
		if len(got) != len(tc.expected) {
			t.Errorf("splitWords(%q): expected %v (len %d), got %v (len %d)",
				tc.input, tc.expected, len(tc.expected), got, len(got))
			continue
		}
		for i := range got {
			if got[i] != tc.expected[i] {
				t.Errorf("splitWords(%q)[%d]: expected %q, got %q",
					tc.input, i, tc.expected[i], got[i])
			}
		}
	}
}
