package version

import "testing"

func TestVersion_String(t *testing.T) {
	tests := []struct {
		v    Version
		want string
	}{
		{Version{1, 2, 3}, "1.2.3"},
		{Version{18, 17, 0}, "18.17.0"},
		{Version{0, 0, 0}, "0.0.0"},
	}

	for _, tt := range tests {
		got := tt.v.String()
		if got != tt.want {
			t.Errorf("Version%v.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

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

func TestParseOptional(t *testing.T) {
	tests := []struct {
		input   string
		want    *Version
		wantErr bool
	}{
		{"", nil, false},
		{"   ", nil, false},
		{"1.2.3", &Version{1, 2, 3}, false},
		{"v1.2", &Version{1, 2, 0}, false},
		{"abc", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseOptional(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOptional(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.want == nil && got != nil {
				t.Errorf("ParseOptional(%q) = %v, want nil", tt.input, got)
			}
			if tt.want != nil && (got == nil || *got != *tt.want) {
				t.Errorf("ParseOptional(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtract(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"v18.17.0", Version{18, 17, 0}, false},
		{"node v18.17.0", Version{18, 17, 0}, false},
		{"Go version 1.21.0 linux/amd64", Version{1, 21, 0}, false},
		{"ffmpeg version 6.0-static", Version{6, 0, 0}, false},
		{"Python 3.11.4", Version{3, 11, 4}, false},
		{"no version here", Version{}, true},
		{"", Version{}, true},

		// Digits in the program name must not win over the real version.
		{"s3cmd version 2.3.0", Version{2, 3, 0}, false},
		{"bzip2, a block-sorting file compressor, version 1.0.8", Version{1, 0, 8}, false},
		{"log4j 2.17.1", Version{2, 17, 1}, false},
		{"x264 0.164.3108", Version{0, 164, 3108}, false},
		{"p7zip Version 16.02", Version{16, 2, 0}, false},
		{"7-Zip 23.01 (x64)", Version{23, 1, 0}, false},
		{"libfoo2 3.4.5", Version{3, 4, 5}, false},

		// A version glued to the name is still the only candidate, so it wins.
		{"go version go1.25.0 darwin/arm64", Version{1, 25, 0}, false},
		{"python3.12.1", Version{3, 12, 1}, false},

		// Trailing build metadata must not outrank the version itself.
		{"Docker version 24.0.7, build afdd53b", Version{24, 0, 7}, false},
		{"git version 2.55.0 (Apple Git-154)", Version{2, 55, 0}, false},
		{"openjdk version \"21.0.1\" 2023-10-17", Version{21, 0, 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Extract(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Extract(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
