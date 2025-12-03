package version

import "testing"

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		a, b Version
		want int
	}{
		{Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{Version{1, 0, 0}, Version{2, 0, 0}, -1},
		{Version{2, 0, 0}, Version{1, 0, 0}, 1},
		{Version{1, 1, 0}, Version{1, 0, 0}, 1},
		{Version{1, 0, 1}, Version{1, 0, 0}, 1},
		{Version{18, 17, 0}, Version{18, 0, 0}, 1},
		{Version{18, 17, 0}, Version{22, 0, 0}, -1},
	}

	for _, tt := range tests {
		got := tt.a.Compare(tt.b)
		if got != tt.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestVersion_LessThan(t *testing.T) {
	tests := []struct {
		a, b Version
		want bool
	}{
		{Version{1, 0, 0}, Version{2, 0, 0}, true},
		{Version{2, 0, 0}, Version{1, 0, 0}, false},
		{Version{1, 0, 0}, Version{1, 0, 0}, false},
	}

	for _, tt := range tests {
		got := tt.a.LessThan(tt.b)
		if got != tt.want {
			t.Errorf("%v.LessThan(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestVersion_GreaterThanOrEqual(t *testing.T) {
	tests := []struct {
		a, b Version
		want bool
	}{
		{Version{1, 0, 0}, Version{1, 0, 0}, true},
		{Version{2, 0, 0}, Version{1, 0, 0}, true},
		{Version{1, 0, 0}, Version{2, 0, 0}, false},
	}

	for _, tt := range tests {
		got := tt.a.GreaterThanOrEqual(tt.b)
		if got != tt.want {
			t.Errorf("%v.GreaterThanOrEqual(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
