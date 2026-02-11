package layout

import "testing"

func TestCalculate_WideScreen(t *testing.T) {
	l := Calculate(160, 40, true)

	if l.SinglePanel {
		t.Error("should not be single panel at 160 cols")
	}
	if l.TwoPanelMode {
		t.Error("should not be two panel mode at 160 cols")
	}
	if l.SidebarWidth == 0 {
		t.Error("sidebar should have width when visible")
	}
	if l.SidebarWidth < minSidebarWidth {
		t.Errorf("sidebar too narrow: %d < %d", l.SidebarWidth, minSidebarWidth)
	}
	if l.SidebarWidth > maxSidebarWidth {
		t.Errorf("sidebar too wide: %d > %d", l.SidebarWidth, maxSidebarWidth)
	}
	total := l.SidebarWidth + l.EditorWidth + l.ResponseWidth
	if total != 160 {
		t.Errorf("panel widths should sum to 160, got %d", total)
	}
}

func TestCalculate_MediumScreen(t *testing.T) {
	l := Calculate(80, 30, true)

	if l.SinglePanel {
		t.Error("should not be single panel at 80 cols")
	}
	if !l.TwoPanelMode {
		t.Error("should be two panel mode at 80 cols")
	}
	if l.SidebarVisible {
		t.Error("sidebar should be hidden in two-panel mode")
	}
}

func TestCalculate_NarrowScreen(t *testing.T) {
	l := Calculate(50, 20, true)

	if !l.SinglePanel {
		t.Error("should be single panel at 50 cols")
	}
}

func TestCalculate_SidebarHidden(t *testing.T) {
	l := Calculate(160, 40, false)

	if l.SidebarWidth != 0 {
		t.Error("sidebar width should be 0 when hidden")
	}
	total := l.EditorWidth + l.ResponseWidth
	if total != 160 {
		t.Errorf("editor+response should sum to 160, got %d", total)
	}
}
