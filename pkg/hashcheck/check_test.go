package hashcheck

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

// mockHashFileOpener implements HashFileOpener for testing.
type mockHashFileOpener struct {
	OpenFunc func(name string) (io.ReadCloser, error)
}

func (m *mockHashFileOpener) Open(name string) (io.ReadCloser, error) {
	return m.OpenFunc(name)
}

// mockReadCloser wraps a string as an io.ReadCloser.
type mockReadCloser struct {
	reader io.Reader
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *mockReadCloser) Close() error {
	return nil
}

func newMockReader(content string) io.ReadCloser {
	return &mockReadCloser{reader: strings.NewReader(content)}
}

// Known hash values for testing (computed with echo -n "test content" | sha256sum etc.)
const (
	// SHA256 of "test content"
	testContentSHA256 = "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
	// SHA384 of "test content"
	testContentSHA384 = "f1c14ae665be79e55b00eedc970704557d72a3021ab3b88ccfdc1b83d1d66c479091e23cfb6021f43b7a1273a6f4a318"
	// SHA512 of "test content"
	testContentSHA512 = "0cbf4caef38047bba9a24e621a961484e5d2a92176a859e7eb27df343dd34eb98d538a6c5f4da1ce302ec250b821cc001e46cc97a704988297185a4df7e99602"
	// SHA1 of "test content"
	testContentSHA1 = "1eebdf4fdc9fc7bf283031b93f9aef3338de9052"
	// MD5 of "test content"
	testContentMD5 = "9473fdd0d880a43c21b7778d34872157"
	// BLAKE2b-256 of "test content"
	testContentBLAKE2b = "8d3bd9bfd005f4179699b9212273284c6de851ae910eee2fd1bb0d96ede65a3d"
	// BLAKE3 of "test content"
	testContentBLAKE3 = "ead3df8af4aece7792496936f83b6b6d191a7f256585ce6b6028db161278017e"
)

func TestHashCheck(t *testing.T) {
	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantName      string
		wantDetailSub string
	}{
		// --- Basic Success Cases ---
		{
			name: "SHA256 hash matches",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA256,
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
			wantName:   "hash: test.txt",
		},
		{
			name: "SHA512 hash matches",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA512,
				Algorithm:    AlgorithmSHA512,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "MD5 hash matches",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentMD5,
				Algorithm:    AlgorithmMD5,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "SHA384 hash matches",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA384,
				Algorithm:    AlgorithmSHA384,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "SHA1 hash matches",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA1,
				Algorithm:    AlgorithmSHA1,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "BLAKE2b-256 hash matches",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentBLAKE2b,
				Algorithm:    AlgorithmBLAKE2b,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "BLAKE3 hash matches",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentBLAKE3,
				Algorithm:    AlgorithmBLAKE3,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "default algorithm is SHA256",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA256,
				// Algorithm not set - should default to SHA256
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --- Auto-detect Tests ---
		{
			name: "auto-detect SHA256 by length (64 chars)",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA256,
				AutoDetect:   true,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "auto-detect SHA512 by length (128 chars)",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA512,
				AutoDetect:   true,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "auto-detect SHA384 by length (96 chars)",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA384,
				AutoDetect:   true,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "auto-detect SHA1 by length (40 chars)",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA1,
				AutoDetect:   true,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "auto-detect MD5 by length (32 chars)",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentMD5,
				AutoDetect:   true,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "auto-detect fails for unrecognized length",
			check: Check{
				File:         "test.txt",
				ExpectedHash: "abcd1234abcd", // 12 chars - not a valid length
				AutoDetect:   true,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "cannot auto-detect algorithm",
		},
		{
			name: "hash comparison is case insensitive",
			check: Check{
				File:         "test.txt",
				ExpectedHash: strings.ToUpper(testContentSHA256),
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --- Hash Mismatch ---
		{
			name: "SHA256 hash mismatch fails",
			check: Check{
				File:         "test.txt",
				ExpectedHash: "0000000000000000000000000000000000000000000000000000000000000000",
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "hash mismatch",
		},
		{
			name: "hash mismatch shows actual hash",
			check: Check{
				File:         "test.txt",
				ExpectedHash: "0000000000000000000000000000000000000000000000000000000000000000",
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: testContentSHA256,
		},

		// --- File Errors ---
		{
			name: "file not found",
			check: Check{
				File:         "nonexistent.txt",
				ExpectedHash: testContentSHA256,
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return nil, os.ErrNotExist
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "not found",
		},
		{
			name: "permission denied",
			check: Check{
				File:         "protected.txt",
				ExpectedHash: testContentSHA256,
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return nil, os.ErrPermission
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "permission denied",
		},
		{
			name: "file read error",
			check: Check{
				File:         "error.txt",
				ExpectedHash: testContentSHA256,
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return nil, errors.New("disk I/O error")
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "failed to open",
		},

		// --- Invalid Input ---
		{
			name: "empty file path",
			check: Check{
				File:         "",
				ExpectedHash: testContentSHA256,
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "file path is required",
		},
		{
			name: "no expected hash provided",
			check: Check{
				File:         "test.txt",
				ExpectedHash: "",
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "expected hash is required",
		},
		{
			name: "invalid hash - non-hex characters",
			check: Check{
				File:         "test.txt",
				ExpectedHash: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "not valid hexadecimal",
		},
		{
			name: "invalid hash - wrong length for SHA256",
			check: Check{
				File:         "test.txt",
				ExpectedHash: "abcd1234", // too short
				Algorithm:    AlgorithmSHA256,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "expected 64 characters",
		},
		{
			name: "invalid hash - wrong length for SHA512",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA256, // 64 chars, but SHA512 needs 128
				Algorithm:    AlgorithmSHA512,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "expected 128 characters",
		},
		{
			name: "invalid hash - wrong length for MD5",
			check: Check{
				File:         "test.txt",
				ExpectedHash: testContentSHA256, // 64 chars, but MD5 needs 32
				Algorithm:    AlgorithmMD5,
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						return newMockReader("test content"), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "expected 32 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v (details: %v)", result.Status, tt.wantStatus, result.Details)
			}
			if tt.wantName != "" && result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			if tt.wantDetailSub != "" {
				allDetails := strings.Join(result.Details, " ")
				if !strings.Contains(allDetails, tt.wantDetailSub) {
					t.Errorf("Details %v should contain %q", result.Details, tt.wantDetailSub)
				}
			}
		})
	}
}

func TestChecksumFileParsing(t *testing.T) {
	tests := []struct {
		name            string
		checksumContent string
		targetFile      string
		wantHash        string
		wantAlgorithm   HashAlgorithm
		wantErr         bool
		wantErrSub      string
	}{
		{
			name:            "GNU format - SHA256",
			checksumContent: testContentSHA256 + "  test.txt\n",
			targetFile:      "test.txt",
			wantHash:        testContentSHA256,
			wantAlgorithm:   AlgorithmSHA256,
		},
		{
			name:            "GNU format - binary mode with asterisk",
			checksumContent: testContentSHA256 + " *test.txt\n",
			targetFile:      "test.txt",
			wantHash:        testContentSHA256,
			wantAlgorithm:   AlgorithmSHA256,
		},
		{
			name:            "GNU format - SHA512 detected by length",
			checksumContent: testContentSHA512 + "  test.txt\n",
			targetFile:      "test.txt",
			wantHash:        testContentSHA512,
			wantAlgorithm:   AlgorithmSHA512,
		},
		{
			name:            "GNU format - MD5 detected by length",
			checksumContent: testContentMD5 + "  test.txt\n",
			targetFile:      "test.txt",
			wantHash:        testContentMD5,
			wantAlgorithm:   AlgorithmMD5,
		},
		{
			name:            "BSD format - SHA256",
			checksumContent: "SHA256 (test.txt) = " + testContentSHA256 + "\n",
			targetFile:      "test.txt",
			wantHash:        testContentSHA256,
			wantAlgorithm:   AlgorithmSHA256,
		},
		{
			name:            "BSD format - SHA512",
			checksumContent: "SHA512 (test.txt) = " + testContentSHA512 + "\n",
			targetFile:      "test.txt",
			wantHash:        testContentSHA512,
			wantAlgorithm:   AlgorithmSHA512,
		},
		{
			name:            "BSD format - MD5",
			checksumContent: "MD5 (test.txt) = " + testContentMD5 + "\n",
			targetFile:      "test.txt",
			wantHash:        testContentMD5,
			wantAlgorithm:   AlgorithmMD5,
		},
		{
			name:            "BSD format - BLAKE2B256",
			checksumContent: "BLAKE2B256 (test.txt) = " + testContentBLAKE2b + "\n",
			targetFile:      "test.txt",
			wantHash:        testContentBLAKE2b,
			wantAlgorithm:   AlgorithmBLAKE2b,
		},
		{
			name:            "BSD format - BLAKE3",
			checksumContent: "BLAKE3 (test.txt) = " + testContentBLAKE3 + "\n",
			targetFile:      "test.txt",
			wantHash:        testContentBLAKE3,
			wantAlgorithm:   AlgorithmBLAKE3,
		},
		{
			name: "multiple entries - finds correct one",
			checksumContent: `abc123  other.txt
` + testContentSHA256 + `  test.txt
def456  another.txt
`,
			targetFile:    "test.txt",
			wantHash:      testContentSHA256,
			wantAlgorithm: AlgorithmSHA256,
		},
		{
			name:            "file not found in checksum file",
			checksumContent: testContentSHA256 + "  other.txt\n",
			targetFile:      "test.txt",
			wantErr:         true,
			wantErrSub:      "not found in checksum file",
		},
		{
			name:            "empty checksum file",
			checksumContent: "",
			targetFile:      "test.txt",
			wantErr:         true,
			wantErrSub:      "not found in checksum file",
		},
		{
			name: "ignores comments and empty lines",
			checksumContent: `# This is a comment
` + testContentSHA256 + `  test.txt

# Another comment
`,
			targetFile:    "test.txt",
			wantHash:      testContentSHA256,
			wantAlgorithm: AlgorithmSHA256,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Check{
				File:         tt.targetFile,
				ChecksumFile: "checksums.txt",
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						if name == "checksums.txt" {
							return newMockReader(tt.checksumContent), nil
						}
						return newMockReader("test content"), nil
					},
				},
			}

			hash, algo, err := c.parseChecksumFile()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("error %q should contain %q", err.Error(), tt.wantErrSub)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if hash != tt.wantHash {
				t.Errorf("hash = %q, want %q", hash, tt.wantHash)
			}
			if algo != tt.wantAlgorithm {
				t.Errorf("algorithm = %q, want %q", algo, tt.wantAlgorithm)
			}
		})
	}
}

func TestChecksumFileIntegration(t *testing.T) {
	t.Run("full check with checksum file", func(t *testing.T) {
		checksumContent := testContentSHA256 + "  test.txt\n"
		fileContent := "test content"

		c := Check{
			File:         "test.txt",
			ChecksumFile: "checksums.txt",
			Opener: &mockHashFileOpener{
				OpenFunc: func(name string) (io.ReadCloser, error) {
					if name == "checksums.txt" {
						return newMockReader(checksumContent), nil
					}
					return newMockReader(fileContent), nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
		}
	})

	t.Run("checksum file not found", func(t *testing.T) {
		c := Check{
			File:         "test.txt",
			ChecksumFile: "nonexistent.txt",
			Opener: &mockHashFileOpener{
				OpenFunc: func(name string) (io.ReadCloser, error) {
					if name == "nonexistent.txt" {
						return nil, os.ErrNotExist
					}
					return newMockReader("test content"), nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusFail {
			t.Errorf("Status = %v, want FAIL", result.Status)
		}
		allDetails := strings.Join(result.Details, " ")
		if !strings.Contains(allDetails, "checksum file") {
			t.Errorf("Details should mention checksum file: %v", result.Details)
		}
	})
}

func TestRealHashFileOpener(t *testing.T) {
	t.Run("opens real file", func(t *testing.T) {
		opener := &RealHashFileOpener{}
		// This file should exist in any Go project
		f, err := opener.Open("check.go")
		if err != nil {
			t.Fatalf("failed to open check.go: %v", err)
		}
		defer func() { _ = f.Close() }()

		// Read a bit to verify it works
		buf := make([]byte, 10)
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("failed to read: %v", err)
		}
		if n == 0 {
			t.Error("read 0 bytes")
		}
	})
}

func TestDetectAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		hashStr  string
		wantAlgo HashAlgorithm
	}{
		{"MD5 (32 chars)", strings.Repeat("a", 32), AlgorithmMD5},
		{"SHA1 (40 chars)", strings.Repeat("b", 40), AlgorithmSHA1},
		{"SHA256 (64 chars)", strings.Repeat("c", 64), AlgorithmSHA256},
		{"SHA384 (96 chars)", strings.Repeat("d", 96), AlgorithmSHA384},
		{"SHA512 (128 chars)", strings.Repeat("e", 128), AlgorithmSHA512},
		{"unknown (48 chars)", strings.Repeat("f", 48), ""},
		{"unknown (100 chars)", strings.Repeat("0", 100), ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectAlgorithm(tt.hashStr)
			if got != tt.wantAlgo {
				t.Errorf("DetectAlgorithm(%d chars) = %q, want %q", len(tt.hashStr), got, tt.wantAlgo)
			}
		})
	}
}
