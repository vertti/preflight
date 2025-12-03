package version

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"1.2.3", Version{1, 2, 3}, false},
		{"v1.2.3", Version{1, 2, 3}, false},
		{"18.17.0", Version{18, 17, 0}, false},
		{"1.2", Version{1, 2, 0}, false},
		{"1", Version{1, 0, 0}, false},
		{"v22", Version{22, 0, 0}, false},
		{"", Version{}, true},
		{"abc", Version{}, true},
		{"1.2.3.4", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtract(t *testing.T) {
	tests := []struct {
		input string
		want  Version
	}{
		{"v18.17.0", Version{18, 17, 0}},
		{"node v18.17.0", Version{18, 17, 0}},
		{"Go version 1.21.0 linux/amd64", Version{1, 21, 0}},
		{"ffmpeg version 6.0-static", Version{6, 0, 0}},
		{"Python 3.11.4", Version{3, 11, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Extract(tt.input)
			if err != nil {
				t.Errorf("Extract(%q) error = %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("Extract(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
