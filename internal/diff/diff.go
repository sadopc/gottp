package diff

// DiffType represents whether a line was added, removed, or unchanged.
type DiffType int

const (
	Same    DiffType = iota
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

// DiffLines computes a line-based diff between two strings using Myers algorithm.
func DiffLines(a, b string) []DiffLine {
	aLines := splitLines(a)
	bLines := splitLines(b)

	// Compute edit script using Myers algorithm
	script := myersDiff(aLines, bLines)
	return script
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

// myersDiff implements the Myers diff algorithm.
func myersDiff(a, b []string) []DiffLine {
	n := len(a)
	m := len(b)

	if n == 0 && m == 0 {
		return nil
	}

	if n == 0 {
		result := make([]DiffLine, m)
		for i, line := range b {
			result[i] = DiffLine{Type: Added, Content: line, OldLine: -1, NewLine: i + 1}
		}
		return result
	}

	if m == 0 {
		result := make([]DiffLine, n)
		for i, line := range a {
			result[i] = DiffLine{Type: Removed, Content: line, OldLine: i + 1, NewLine: -1}
		}
		return result
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
				return buildResult(trace, a, b, max)
			}
		}
	}

	return buildResult(trace, a, b, max)
}

func buildResult(trace [][]int, a, b []string, max int) []DiffLine {
	n := len(a)
	m := len(b)

	// Backtrack to find the path
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

		// Diagonal (same lines)
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

	// Build diff lines
	var result []DiffLine
	for _, p := range path {
		if p.x == -1 {
			// Added
			result = append(result, DiffLine{
				Type: Added, Content: b[p.y], OldLine: -1, NewLine: p.y + 1,
			})
		} else if p.y == -1 {
			// Removed
			result = append(result, DiffLine{
				Type: Removed, Content: a[p.x], OldLine: p.x + 1, NewLine: -1,
			})
		} else {
			// Same
			result = append(result, DiffLine{
				Type: Same, Content: a[p.x], OldLine: p.x + 1, NewLine: p.y + 1,
			})
		}
	}

	return result
}
