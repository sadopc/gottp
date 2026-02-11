package msgs

import "testing"

func TestAppModeString(t *testing.T) {
	tests := []struct {
		name string
		mode AppMode
		want string
	}{
		{name: "normal", mode: ModeNormal, want: "NORMAL"},
		{name: "insert", mode: ModeInsert, want: "INSERT"},
		{name: "command", mode: ModeCommandPalette, want: "COMMAND"},
		{name: "jump", mode: ModeJump, want: "JUMP"},
		{name: "modal", mode: ModeModal, want: "MODAL"},
		{name: "search", mode: ModeSearch, want: "SEARCH"},
		{name: "unknown", mode: AppMode(999), want: "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
