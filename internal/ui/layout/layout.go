package layout

// PanelLayout holds calculated dimensions for the three-panel layout.
type PanelLayout struct {
	Width  int
	Height int

	SidebarWidth  int
	EditorWidth   int
	ResponseWidth int

	ContentHeight int // height minus tab bar and status bar

	SidebarVisible bool
	TwoPanelMode   bool
	SinglePanel    bool
}

const (
	tabBarHeight    = 1
	statusBarHeight = 1
	minSidebarWidth = 20
	maxSidebarWidth = 35
)

// Calculate computes the panel layout from terminal dimensions.
func Calculate(width, height int, sidebarVisible bool) PanelLayout {
	l := PanelLayout{
		Width:          width,
		Height:         height,
		SidebarVisible: sidebarVisible,
		ContentHeight:  height - tabBarHeight - statusBarHeight,
	}

	if l.ContentHeight < 1 {
		l.ContentHeight = 1
	}

	// Responsive breakpoints
	switch {
	case width < 60:
		l.SinglePanel = true
		l.SidebarVisible = false
		l.EditorWidth = width
		l.ResponseWidth = width
	case width < 100:
		l.TwoPanelMode = true
		l.SidebarVisible = false
		half := width / 2
		l.EditorWidth = half
		l.ResponseWidth = width - half
	default:
		if sidebarVisible {
			l.SidebarWidth = clamp(width/5, minSidebarWidth, maxSidebarWidth)
			remaining := width - l.SidebarWidth
			l.EditorWidth = remaining / 2
			l.ResponseWidth = remaining - l.EditorWidth
		} else {
			half := width / 2
			l.EditorWidth = half
			l.ResponseWidth = width - half
		}
	}

	return l
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
