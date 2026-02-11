package diff

import "unicode"

// DiffType represents whether a line was added, removed, or unchanged.
type DiffType int

const (
	Same DiffType = iota
	Added
	Removed
)

// DiffLine represents a single line in the diff output.
type DiffLine struct {
	Type    DiffType
	Content string
	OldLine int // line number in old text (-1 if added)
	NewLine int // line number in new text (-1 if removed)
}

// WordDiff represents a word-level change within a line.
type WordDiff struct {
	Type    DiffType
	Content string
}

// RichDiffLine is a diff line with optional word-level detail.
type RichDiffLine struct {
	DiffLine
	Words []WordDiff // non-nil for modified lines (adjacent remove+add pairs)
}

// DiffLines computes a line-based diff between two strings using Myers algorithm.
func DiffLines(a, b string) []DiffLine {
	aLines := splitLines(a)
	bLines := splitLines(b)

	edits := myersDiffGeneric(aLines, bLines)
	return editsToDiffLines(edits, aLines, bLines)
}

// DiffWords computes a word-level diff between two strings.
// It splits by whitespace boundaries and diffs the resulting tokens.
func DiffWords(a, b string) []WordDiff {
	aWords := splitWords(a)
	bWords := splitWords(b)

	edits := myersDiffGeneric(aWords, bWords)
	return editsToWordDiffs(edits, aWords, bWords)
}

// DiffLinesWithWords computes line-level diff then enriches modified lines
// with word-level detail. Consecutive Removed+Added pairs are treated as
// modifications and receive word-level diffs in their Words field.
func DiffLinesWithWords(a, b string) []RichDiffLine {
	lines := DiffLines(a, b)

	result := make([]RichDiffLine, len(lines))
	for i, l := range lines {
		result[i] = RichDiffLine{DiffLine: l}
	}

	// Scan for consecutive Removed+Added pairs (modifications).
	// Collect contiguous runs of Removed followed by contiguous runs of Added.
	i := 0
	for i < len(result) {
		// Find a run of Removed lines.
		remStart := i
		for i < len(result) && result[i].Type == Removed {
			i++
		}
		remEnd := i

		// Find a run of Added lines immediately following.
		addStart := i
		for i < len(result) && result[i].Type == Added {
			i++
		}
		addEnd := i

		remCount := remEnd - remStart
		addCount := addEnd - addStart

		if remCount == 0 || addCount == 0 {
			// Not a modification pair; skip.
			if remCount == 0 && addCount == 0 {
				i++ // advance past Same line
			}
			continue
		}

		// Pair up removed and added lines 1:1 up to the shorter count.
		pairs := remCount
		if addCount < pairs {
			pairs = addCount
		}
		for j := 0; j < pairs; j++ {
			words := DiffWords(result[remStart+j].Content, result[addStart+j].Content)
			result[remStart+j].Words = words
			result[addStart+j].Words = words
		}
	}

	return result
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// splitWords splits a string into tokens preserving whitespace as separate tokens.
// For example: "hello world  foo" → ["hello", " ", "world", "  ", "foo"]
func splitWords(s string) []string {
	if s == "" {
		return nil
	}

	var tokens []string
	i := 0
	runes := []rune(s)
	for i < len(runes) {
		if unicode.IsSpace(runes[i]) {
			// Collect contiguous whitespace.
			start := i
			for i < len(runes) && unicode.IsSpace(runes[i]) {
				i++
			}
			tokens = append(tokens, string(runes[start:i]))
		} else {
			// Collect contiguous non-whitespace.
			start := i
			for i < len(runes) && !unicode.IsSpace(runes[i]) {
				i++
			}
			tokens = append(tokens, string(runes[start:i]))
		}
	}
	return tokens
}

// editOp represents a single edit operation from the Myers diff.
type editOp int

const (
	opEqual  editOp = iota // diagonal move (tokens match)
	opInsert               // vertical move (token in b only)
	opDelete               // horizontal move (token in a only)
)

// edit represents a single step in the edit script.
type edit struct {
	op   editOp
	aIdx int // index into a (-1 if insert)
	bIdx int // index into b (-1 if delete)
}

// myersDiffGeneric implements the Myers diff algorithm on generic string slices.
// It returns a sequence of edit operations.
func myersDiffGeneric(a, b []string) []edit {
	n := len(a)
	m := len(b)

	if n == 0 && m == 0 {
		return nil
	}

	if n == 0 {
		edits := make([]edit, m)
		for i := range b {
			edits[i] = edit{op: opInsert, aIdx: -1, bIdx: i}
		}
		return edits
	}

	if m == 0 {
		edits := make([]edit, n)
		for i := range a {
			edits[i] = edit{op: opDelete, aIdx: i, bIdx: -1}
		}
		return edits
	}

	max := n + m
	v := make([]int, 2*max+1)
	var trace [][]int

	for d := 0; d <= max; d++ {
		snapshot := make([]int, len(v))
		copy(snapshot, v)
		trace = append(trace, snapshot)

		for k := -d; k <= d; k += 2 {
			var x int
			idx := k + max
			if k == -d || (k != d && v[idx-1] < v[idx+1]) {
				x = v[idx+1]
			} else {
				x = v[idx-1] + 1
			}
			y := x - k

			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}

			v[idx] = x

			if x >= n && y >= m {
				return backtrack(trace, a, b, max)
			}
		}
	}

	return backtrack(trace, a, b, max)
}

// backtrack reconstructs the edit script from the Myers trace.
func backtrack(trace [][]int, a, b []string, max int) []edit {
	n := len(a)
	m := len(b)

	type point struct{ x, y int }
	var path []point

	x, y := n, m
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		k := x - y
		idx := k + max

		var prevK int
		if k == -d || (k != d && v[idx-1] < v[idx+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := v[prevK+max]
		prevY := prevX - prevK

		// Diagonal (equal tokens)
		for x > prevX && y > prevY {
			x--
			y--
			path = append(path, point{x, y})
		}

		if d > 0 {
			if x == prevX {
				// Insert (added in b)
				y--
				path = append(path, point{-1, y})
			} else {
				// Delete (removed from a)
				x--
				path = append(path, point{x, -1})
			}
		}

		x = prevX
		y = prevY
	}

	// Reverse path
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	// Convert path to edit operations
	edits := make([]edit, len(path))
	for i, p := range path {
		switch {
		case p.x == -1:
			edits[i] = edit{op: opInsert, aIdx: -1, bIdx: p.y}
		case p.y == -1:
			edits[i] = edit{op: opDelete, aIdx: p.x, bIdx: -1}
		default:
			edits[i] = edit{op: opEqual, aIdx: p.x, bIdx: p.y}
		}
	}

	return edits
}

// editsToDiffLines converts generic edit operations to DiffLine results.
func editsToDiffLines(edits []edit, a, b []string) []DiffLine {
	result := make([]DiffLine, 0, len(edits))
	for _, e := range edits {
		switch e.op {
		case opInsert:
			result = append(result, DiffLine{
				Type: Added, Content: b[e.bIdx], OldLine: -1, NewLine: e.bIdx + 1,
			})
		case opDelete:
			result = append(result, DiffLine{
				Type: Removed, Content: a[e.aIdx], OldLine: e.aIdx + 1, NewLine: -1,
			})
		case opEqual:
			result = append(result, DiffLine{
				Type: Same, Content: a[e.aIdx], OldLine: e.aIdx + 1, NewLine: e.bIdx + 1,
			})
		}
	}
	return result
}

// editsToWordDiffs converts generic edit operations to WordDiff results.
func editsToWordDiffs(edits []edit, a, b []string) []WordDiff {
	result := make([]WordDiff, 0, len(edits))
	for _, e := range edits {
		switch e.op {
		case opInsert:
			result = append(result, WordDiff{Type: Added, Content: b[e.bIdx]})
		case opDelete:
			result = append(result, WordDiff{Type: Removed, Content: a[e.aIdx]})
		case opEqual:
			result = append(result, WordDiff{Type: Same, Content: a[e.aIdx]})
		}
	}
	return result
}
