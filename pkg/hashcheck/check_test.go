package hashcheck

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vertti/preflight/pkg/check"
)

type mockHashFileOpener struct {
	OpenFunc func(name string) (io.ReadCloser, error)
}

func (m *mockHashFileOpener) Open(name string) (io.ReadCloser, error) {
	return m.OpenFunc(name)
}

type mockReadCloser struct{ reader io.Reader }

func (m *mockReadCloser) Read(p []byte) (n int, err error) { return m.reader.Read(p) }
func (m *mockReadCloser) Close() error                     { return nil }

func mockReader(content string) io.ReadCloser {
	return &mockReadCloser{reader: strings.NewReader(content)}
}

func opener(content string) *mockHashFileOpener {
	return &mockHashFileOpener{OpenFunc: func(string) (io.ReadCloser, error) { return mockReader(content), nil }}
}

func openerErr(err error) *mockHashFileOpener {
	return &mockHashFileOpener{OpenFunc: func(string) (io.ReadCloser, error) { return nil, err }}
}

const (
	testContentSHA256 = "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
	testContentSHA384 = "f1c14ae665be79e55b00eedc970704557d72a3021ab3b88ccfdc1b83d1d66c479091e23cfb6021f43b7a1273a6f4a318"
	testContentSHA512 = "0cbf4caef38047bba9a24e621a961484e5d2a92176a859e7eb27df343dd34eb98d538a6c5f4da1ce302ec250b821cc001e46cc97a704988297185a4df7e99602"
	testContentSHA1   = "1eebdf4fdc9fc7bf283031b93f9aef3338de9052"
	testContentMD5    = "9473fdd0d880a43c21b7778d34872157"
)

func TestHashCheck(t *testing.T) {
	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantDetailSub string
	}{
		// Algorithm tests
		{"SHA256 matches", Check{File: "test.txt", ExpectedHash: testContentSHA256, Algorithm: AlgorithmSHA256, Opener: opener("test content")}, check.StatusOK, ""},
		{"SHA512 matches", Check{File: "test.txt", ExpectedHash: testContentSHA512, Algorithm: AlgorithmSHA512, Opener: opener("test content")}, check.StatusOK, ""},
		{"MD5 matches", Check{File: "test.txt", ExpectedHash: testContentMD5, Algorithm: AlgorithmMD5, Opener: opener("test content")}, check.StatusOK, ""},
		{"SHA384 matches", Check{File: "test.txt", ExpectedHash: testContentSHA384, Algorithm: AlgorithmSHA384, Opener: opener("test content")}, check.StatusOK, ""},
		{"SHA1 matches", Check{File: "test.txt", ExpectedHash: testContentSHA1, Algorithm: AlgorithmSHA1, Opener: opener("test content")}, check.StatusOK, ""},
		{"default algorithm is SHA256", Check{File: "test.txt", ExpectedHash: testContentSHA256, Opener: opener("test content")}, check.StatusOK, ""},

		// Auto-detect
		{"auto-detect SHA256", Check{File: "test.txt", ExpectedHash: testContentSHA256, AutoDetect: true, Opener: opener("test content")}, check.StatusOK, ""},
		{"auto-detect SHA512", Check{File: "test.txt", ExpectedHash: testContentSHA512, AutoDetect: true, Opener: opener("test content")}, check.StatusOK, ""},
		{"auto-detect SHA384", Check{File: "test.txt", ExpectedHash: testContentSHA384, AutoDetect: true, Opener: opener("test content")}, check.StatusOK, ""},
		{"auto-detect SHA1", Check{File: "test.txt", ExpectedHash: testContentSHA1, AutoDetect: true, Opener: opener("test content")}, check.StatusOK, ""},
		{"auto-detect MD5", Check{File: "test.txt", ExpectedHash: testContentMD5, AutoDetect: true, Opener: opener("test content")}, check.StatusOK, ""},
		{"auto-detect fails unrecognized length", Check{File: "test.txt", ExpectedHash: "abcd1234abcd", AutoDetect: true, Opener: opener("test content")}, check.StatusFail, "cannot auto-detect"},
		{"case insensitive", Check{File: "test.txt", ExpectedHash: strings.ToUpper(testContentSHA256), Algorithm: AlgorithmSHA256, Opener: opener("test content")}, check.StatusOK, ""},

		// Mismatch
		{"hash mismatch fails", Check{File: "test.txt", ExpectedHash: strings.Repeat("0", 64), Algorithm: AlgorithmSHA256, Opener: opener("test content")}, check.StatusFail, "hash mismatch"},
		{"mismatch shows actual hash", Check{File: "test.txt", ExpectedHash: strings.Repeat("0", 64), Algorithm: AlgorithmSHA256, Opener: opener("test content")}, check.StatusFail, testContentSHA256},

		// File errors
		{"file not found", Check{File: "nonexistent.txt", ExpectedHash: testContentSHA256, Algorithm: AlgorithmSHA256, Opener: openerErr(os.ErrNotExist)}, check.StatusFail, "not found"},
		{"permission denied", Check{File: "protected.txt", ExpectedHash: testContentSHA256, Algorithm: AlgorithmSHA256, Opener: openerErr(os.ErrPermission)}, check.StatusFail, "permission denied"},
		{"file read error", Check{File: "error.txt", ExpectedHash: testContentSHA256, Algorithm: AlgorithmSHA256, Opener: openerErr(errors.New("disk I/O error"))}, check.StatusFail, "failed to open"},

		// Invalid input
		{"empty file path", Check{File: "", ExpectedHash: testContentSHA256}, check.StatusFail, "file path is required"},
		{"no expected hash", Check{File: "test.txt", ExpectedHash: "", Opener: opener("test content")}, check.StatusFail, "expected hash is required"},
		{"invalid hash non-hex", Check{File: "test.txt", ExpectedHash: strings.Repeat("z", 64), Algorithm: AlgorithmSHA256, Opener: opener("test content")}, check.StatusFail, "not valid hexadecimal"},
		{"wrong length SHA256", Check{File: "test.txt", ExpectedHash: "abcd1234", Algorithm: AlgorithmSHA256, Opener: opener("test content")}, check.StatusFail, "expected 64"},
		{"wrong length SHA512", Check{File: "test.txt", ExpectedHash: testContentSHA256, Algorithm: AlgorithmSHA512, Opener: opener("test content")}, check.StatusFail, "expected 128"},
		{"wrong length MD5", Check{File: "test.txt", ExpectedHash: testContentSHA256, Algorithm: AlgorithmMD5, Opener: opener("test content")}, check.StatusFail, "expected 32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			if tt.wantDetailSub != "" {
				assert.Contains(t, strings.Join(result.Details, " "), tt.wantDetailSub)
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
		wantErr         string
	}{
		// GNU format
		{"GNU SHA256", testContentSHA256 + "  test.txt\n", "test.txt", testContentSHA256, AlgorithmSHA256, ""},
		{"GNU binary mode", testContentSHA256 + " *test.txt\n", "test.txt", testContentSHA256, AlgorithmSHA256, ""},
		{"GNU SHA512", testContentSHA512 + "  test.txt\n", "test.txt", testContentSHA512, AlgorithmSHA512, ""},
		{"GNU MD5", testContentMD5 + "  test.txt\n", "test.txt", testContentMD5, AlgorithmMD5, ""},
		{"GNU SHA384", testContentSHA384 + "  test.txt\n", "test.txt", testContentSHA384, AlgorithmSHA384, ""},
		{"GNU SHA1", testContentSHA1 + "  test.txt\n", "test.txt", testContentSHA1, AlgorithmSHA1, ""},

		// BSD format
		{"BSD SHA256", "SHA256 (test.txt) = " + testContentSHA256 + "\n", "test.txt", testContentSHA256, AlgorithmSHA256, ""},
		{"BSD SHA512", "SHA512 (test.txt) = " + testContentSHA512 + "\n", "test.txt", testContentSHA512, AlgorithmSHA512, ""},
		{"BSD MD5", "MD5 (test.txt) = " + testContentMD5 + "\n", "test.txt", testContentMD5, AlgorithmMD5, ""},
		{"BSD SHA384", "SHA384 (test.txt) = " + testContentSHA384 + "\n", "test.txt", testContentSHA384, AlgorithmSHA384, ""},
		{"BSD SHA1", "SHA1 (test.txt) = " + testContentSHA1 + "\n", "test.txt", testContentSHA1, AlgorithmSHA1, ""},

		// Multiple entries
		{"multiple entries", "abc123  other.txt\n" + testContentSHA256 + "  test.txt\ndef456  another.txt\n", "test.txt", testContentSHA256, AlgorithmSHA256, ""},
		{"ignores comments", "# comment\n" + testContentSHA256 + "  test.txt\n\n# another\n", "test.txt", testContentSHA256, AlgorithmSHA256, ""},

		// Errors
		{"file not found in checksum", testContentSHA256 + "  other.txt\n", "test.txt", "", "", "not found in checksum"},
		{"empty checksum file", "", "test.txt", "", "", "not found in checksum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Check{
				File:         tt.targetFile,
				ChecksumFile: "checksums.txt",
				Opener: &mockHashFileOpener{
					OpenFunc: func(name string) (io.ReadCloser, error) {
						if name == "checksums.txt" {
							return mockReader(tt.checksumContent), nil
						}
						return mockReader("test content"), nil
					},
				},
			}

			hash, algo, err := c.parseChecksumFile()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHash, hash)
			assert.Equal(t, tt.wantAlgorithm, algo)
		})
	}
}

func TestChecksumFileIntegration(t *testing.T) {
	t.Run("full check with checksum file", func(t *testing.T) {
		c := Check{
			File:         "test.txt",
			ChecksumFile: "checksums.txt",
			Opener: &mockHashFileOpener{
				OpenFunc: func(name string) (io.ReadCloser, error) {
					if name == "checksums.txt" {
						return mockReader(testContentSHA256 + "  test.txt\n"), nil
					}
					return mockReader("test content"), nil
				},
			},
		}
		result := c.Run()
		assert.Equal(t, check.StatusOK, result.Status, "details: %v", result.Details)
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
					return mockReader("test content"), nil
				},
			},
		}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.Contains(t, strings.Join(result.Details, " "), "checksum file")
	})
}

func TestRealHashFileOpener(t *testing.T) {
	opener := &RealHashFileOpener{}
	f, err := opener.Open("check.go")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	buf := make([]byte, 10)
	n, err := f.Read(buf)
	if err != nil {
		require.ErrorIs(t, err, io.EOF, "unexpected read error")
	}
	assert.Greater(t, n, 0)
}

func TestDetectAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		hashLen  int
		wantAlgo HashAlgorithm
	}{
		{"MD5", 32, AlgorithmMD5},
		{"SHA1", 40, AlgorithmSHA1},
		{"SHA256", 64, AlgorithmSHA256},
		{"SHA384", 96, AlgorithmSHA384},
		{"SHA512", 128, AlgorithmSHA512},
		{"unknown 48", 48, ""},
		{"unknown 100", 100, ""},
		{"empty", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectAlgorithm(strings.Repeat("a", tt.hashLen))
			assert.Equal(t, tt.wantAlgo, got)
		})
	}
}

func TestHashAlgorithmNewHasher(t *testing.T) {
	tests := []struct {
		algo     HashAlgorithm
		wantSize int
	}{
		{AlgorithmSHA256, 32},
		{AlgorithmSHA384, 48},
		{AlgorithmSHA512, 64},
		{AlgorithmSHA1, 20},
		{AlgorithmMD5, 16},
		{HashAlgorithm("unknown"), 32},
		{HashAlgorithm(""), 32},
	}

	for _, tt := range tests {
		t.Run(string(tt.algo), func(t *testing.T) {
			assert.Equal(t, tt.wantSize, tt.algo.NewHasher().Size())
		})
	}
}

func TestHashAlgorithmExpectedHexLength(t *testing.T) {
	tests := []struct {
		algo       HashAlgorithm
		wantLength int
	}{
		{AlgorithmSHA256, 64},
		{AlgorithmSHA384, 96},
		{AlgorithmSHA512, 128},
		{AlgorithmSHA1, 40},
		{AlgorithmMD5, 32},
		{HashAlgorithm("unknown"), 64},
		{HashAlgorithm(""), 64},
	}

	for _, tt := range tests {
		t.Run(string(tt.algo), func(t *testing.T) {
			assert.Equal(t, tt.wantLength, tt.algo.ExpectedHexLength())
		})
	}
}
