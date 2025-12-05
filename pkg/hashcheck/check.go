package hashcheck

import (
	"bufio"
	"crypto/md5" //nolint:gosec // MD5 support is intentional for legacy use
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vertti/preflight/pkg/check"
)

// FileOpener abstracts file operations for testability.
type FileOpener interface {
	Open(name string) (io.ReadCloser, error)
}

// RealFileOpener implements FileOpener using the real filesystem.
type RealFileOpener struct{}

// Open opens a file for reading.
func (r *RealFileOpener) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

// HashAlgorithm represents supported hash algorithms.
type HashAlgorithm string

const (
	AlgorithmSHA256 HashAlgorithm = "sha256"
	AlgorithmSHA512 HashAlgorithm = "sha512"
	AlgorithmMD5    HashAlgorithm = "md5"
)

// Hash lengths in hex characters
const (
	SHA256HexLen = 64
	SHA512HexLen = 128
	MD5HexLen    = 32
)

// Check verifies a file's checksum against an expected hash.
type Check struct {
	File         string        // path to file to verify (required)
	ExpectedHash string        // expected hash value (required unless ChecksumFile is set)
	Algorithm    HashAlgorithm // hash algorithm (default: sha256)
	ChecksumFile string        // path to checksum file (BSD or GNU format)
	Opener       FileOpener    // injected for testing
}

// Run executes the hash verification check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("hash: %s", c.File),
	}

	// Validate file path
	if c.File == "" {
		return result.Failf("file path is required")
	}

	// Set default algorithm
	algorithm := c.Algorithm
	if algorithm == "" {
		algorithm = AlgorithmSHA256
	}

	// Get expected hash - either from direct input or checksum file
	expectedHash := c.ExpectedHash
	if c.ChecksumFile != "" {
		var err error
		expectedHash, algorithm, err = c.parseChecksumFile()
		if err != nil {
			return result.Failf("%v", err)
		}
	}

	// Validate expected hash
	if expectedHash == "" {
		return result.Failf("expected hash is required (use --sha256, --sha512, --md5, or --checksum-file)")
	}

	// Normalize hash to lowercase
	expectedHash = strings.ToLower(expectedHash)

	// Validate hash format
	if err := validateHashFormat(expectedHash, algorithm); err != nil {
		return result.Failf("invalid hash: %v", err)
	}

	// Initialize opener if not injected
	opener := c.Opener
	if opener == nil {
		opener = &RealFileOpener{}
	}

	// Open file
	f, err := opener.Open(c.File)
	if err != nil {
		if os.IsNotExist(err) {
			return result.Failf("file not found")
		}
		if os.IsPermission(err) {
			return result.Failf("permission denied")
		}
		return result.Failf("failed to open file: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Compute hash
	actualHash, err := computeHash(f, algorithm)
	if err != nil {
		return result.Failf("failed to compute hash: %v", err)
	}

	// Compare hashes (case-insensitive, both normalized to lowercase)
	if actualHash != expectedHash {
		return result.Failf("hash mismatch\n       expected: %s\n       actual:   %s", expectedHash, actualHash)
	}

	// Success
	result.Status = check.StatusOK
	result.AddDetailf("algorithm: %s", algorithm)
	result.AddDetailf("hash: %s", actualHash)
	return result
}

// validateHashFormat validates that the hash is valid hex with correct length.
func validateHashFormat(hashStr string, algorithm HashAlgorithm) error {
	// Check hex characters
	if _, err := hex.DecodeString(hashStr); err != nil {
		return fmt.Errorf("not valid hexadecimal")
	}

	// Check length
	expectedLen := getHashLength(algorithm)
	if len(hashStr) != expectedLen {
		return fmt.Errorf("expected %d characters for %s, got %d", expectedLen, algorithm, len(hashStr))
	}

	return nil
}

// getHashLength returns the expected hex string length for an algorithm.
func getHashLength(algorithm HashAlgorithm) int {
	switch algorithm {
	case AlgorithmSHA256:
		return SHA256HexLen
	case AlgorithmSHA512:
		return SHA512HexLen
	case AlgorithmMD5:
		return MD5HexLen
	default:
		return SHA256HexLen
	}
}

// computeHash computes the hash of a reader using the specified algorithm.
func computeHash(r io.Reader, algorithm HashAlgorithm) (string, error) {
	var h hash.Hash
	switch algorithm {
	case AlgorithmSHA256:
		h = sha256.New()
	case AlgorithmSHA512:
		h = sha512.New()
	case AlgorithmMD5:
		h = md5.New() //nolint:gosec // MD5 support is intentional for legacy use
	default:
		h = sha256.New()
	}

	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// BSD format regex: SHA256 (filename) = hash
var bsdFormatRegex = regexp.MustCompile(`^(SHA256|SHA512|MD5)\s+\((.+)\)\s*=\s*([a-fA-F0-9]+)$`)

// parseChecksumFile reads and parses a checksum file to find the hash for c.File.
func (c *Check) parseChecksumFile() (string, HashAlgorithm, error) {
	opener := c.Opener
	if opener == nil {
		opener = &RealFileOpener{}
	}

	f, err := opener.Open(c.ChecksumFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to open checksum file: %v", err)
	}
	defer func() { _ = f.Close() }()

	targetFile := filepath.Base(c.File)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try BSD format first: SHA256 (filename) = hash
		if matches := bsdFormatRegex.FindStringSubmatch(line); matches != nil {
			algoStr, filename, hashValue := matches[1], matches[2], matches[3]
			if filename == targetFile || filename == c.File {
				algo := AlgorithmSHA256
				switch algoStr {
				case "SHA512":
					algo = AlgorithmSHA512
				case "MD5":
					algo = AlgorithmMD5
				}
				return hashValue, algo, nil
			}
			continue
		}

		// Try GNU format: hash  filename or hash *filename
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			hashValue := parts[0]
			filename := parts[len(parts)-1]
			// Remove leading * from binary mode indicator
			filename = strings.TrimPrefix(filename, "*")

			if filename == targetFile || filename == c.File {
				// Determine algorithm from hash length
				algo := AlgorithmSHA256
				switch len(hashValue) {
				case SHA512HexLen:
					algo = AlgorithmSHA512
				case MD5HexLen:
					algo = AlgorithmMD5
				}
				return hashValue, algo, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("error reading checksum file: %v", err)
	}

	return "", "", fmt.Errorf("file %q not found in checksum file", targetFile)
}
