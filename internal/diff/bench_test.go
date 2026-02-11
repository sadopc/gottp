package diff

import (
	"fmt"
	"strings"
	"testing"
)

// generateLines creates n lines with the given prefix and line numbers.
func generateLines(prefix string, n int) string {
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		lines[i] = fmt.Sprintf("%s line %d", prefix, i)
	}
	return strings.Join(lines, "\n")
}

// generatePartiallyDifferent creates two texts of size n where roughly pct% of
// lines differ (every nth line is changed in the second text).
func generatePartiallyDifferent(n int, changeEvery int) (string, string) {
	aLines := make([]string, n)
	bLines := make([]string, n)
	for i := 0; i < n; i++ {
		aLines[i] = fmt.Sprintf("line %d content", i)
		if i%changeEvery == 0 {
			bLines[i] = fmt.Sprintf("modified line %d content", i)
		} else {
			bLines[i] = aLines[i]
		}
	}
	return strings.Join(aLines, "\n"), strings.Join(bLines, "\n")
}

func BenchmarkDiffLinesIdentical(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Lines_%d", n), func(b *testing.B) {
			text := generateLines("identical", n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = DiffLines(text, text)
			}
		})
	}
}

func BenchmarkDiffLinesCompletelyDifferent(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Lines_%d", n), func(b *testing.B) {
			a := generateLines("old", n)
			bText := generateLines("new", n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = DiffLines(a, bText)
			}
		})
	}
}

func BenchmarkDiffLinesPartiallySimilar(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Lines_%d/10pct_changed", n), func(b *testing.B) {
			a, bText := generatePartiallyDifferent(n, 10) // ~10% changed
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = DiffLines(a, bText)
			}
		})

		b.Run(fmt.Sprintf("Lines_%d/25pct_changed", n), func(b *testing.B) {
			a, bText := generatePartiallyDifferent(n, 4) // ~25% changed
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = DiffLines(a, bText)
			}
		})

		b.Run(fmt.Sprintf("Lines_%d/50pct_changed", n), func(b *testing.B) {
			a, bText := generatePartiallyDifferent(n, 2) // ~50% changed
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = DiffLines(a, bText)
			}
		})
	}
}

func BenchmarkDiffLinesEmpty(b *testing.B) {
	b.Run("BothEmpty", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = DiffLines("", "")
		}
	})

	b.Run("LeftEmpty/10_lines", func(b *testing.B) {
		text := generateLines("added", 10)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = DiffLines("", text)
		}
	})

	b.Run("LeftEmpty/100_lines", func(b *testing.B) {
		text := generateLines("added", 100)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = DiffLines("", text)
		}
	})

	b.Run("RightEmpty/10_lines", func(b *testing.B) {
		text := generateLines("removed", 10)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = DiffLines(text, "")
		}
	})

	b.Run("RightEmpty/100_lines", func(b *testing.B) {
		text := generateLines("removed", 100)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = DiffLines(text, "")
		}
	})
}

func BenchmarkDiffLinesInsertionAtEnd(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Lines_%d", n), func(b *testing.B) {
			base := generateLines("line", n)
			withExtra := base + "\nnew line appended"
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = DiffLines(base, withExtra)
			}
		})
	}
}

func BenchmarkDiffLinesSingleLineChange(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Lines_%d", n), func(b *testing.B) {
			aLines := make([]string, n)
			bLines := make([]string, n)
			for i := 0; i < n; i++ {
				aLines[i] = fmt.Sprintf("line %d", i)
				bLines[i] = aLines[i]
			}
			// Change only the middle line
			mid := n / 2
			bLines[mid] = fmt.Sprintf("CHANGED line %d", mid)
			a := strings.Join(aLines, "\n")
			bText := strings.Join(bLines, "\n")
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = DiffLines(a, bText)
			}
		})
	}
}

func BenchmarkSplitLines(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"10_lines", 10},
		{"100_lines", 100},
		{"1000_lines", 1000},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			text := generateLines("line", s.n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = splitLines(text)
			}
		})
	}
}
