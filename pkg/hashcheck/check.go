package hashcheck

import (
	"bufio"
	"crypto/md5"  //nolint:gosec // MD5 support is intentional for legacy use
	"crypto/sha1" //nolint:gosec // SHA1 support is intentional for legacy use
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

// HashFileOpener abstracts file operations for testability.
type HashFileOpener interface {
	Open(name string) (io.ReadCloser, error)
}

// RealHashFileOpener implements HashFileOpener using the real filesystem.
type RealHashFileOpener struct{}

func (r *RealHashFileOpener) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

// HashAlgorithm represents supported hash algorithms.
type HashAlgorithm string

const (
	AlgorithmSHA256 HashAlgorithm = "sha256"
	AlgorithmSHA384 HashAlgorithm = "sha384"
	AlgorithmSHA512 HashAlgorithm = "sha512"
	AlgorithmSHA1   HashAlgorithm = "sha1"
	AlgorithmMD5    HashAlgorithm = "md5"
)

func (a HashAlgorithm) NewHasher() hash.Hash {
	switch a {
	case AlgorithmSHA384:
		return sha512.New384()
	case AlgorithmSHA512:
		return sha512.New()
	case AlgorithmSHA1:
		return sha1.New() //nolint:gosec // SHA1 support is intentional for legacy use
	case AlgorithmMD5:
		return md5.New() //nolint:gosec // MD5 support is intentional for legacy use
	default:
		return sha256.New()
	}
}

func (a HashAlgorithm) ExpectedHexLength() int {
	switch a {
	case AlgorithmSHA512:
		return 128
	case AlgorithmSHA384:
		return 96
	case AlgorithmSHA1:
		return 40
	case AlgorithmMD5:
		return 32
	default:
		return 64 // SHA256
	}
}

// Check verifies a file's checksum against an expected hash.
type Check struct {
	File         string
	ExpectedHash string
	Algorithm    HashAlgorithm
	ChecksumFile string
	Opener       HashFileOpener
}

func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("hash: %s", c.File),
	}

	if c.File == "" {
		return result.Failf("file path is required")
	}

	algorithm := c.Algorithm
	if algorithm == "" {
		algorithm = AlgorithmSHA256
	}

	expectedHash := c.ExpectedHash
	if c.ChecksumFile != "" {
		var err error
		expectedHash, algorithm, err = c.parseChecksumFile()
		if err != nil {
			return result.Failf("%v", err)
		}
	}

	if expectedHash == "" {
		return result.Failf("expected hash is required (use --sha256, --sha384, --sha512, --sha1, --md5, or --checksum-file)")
	}

	expectedHash = strings.ToLower(expectedHash)

	if err := c.validateHash(expectedHash, algorithm); err != nil {
		return result.Failf("invalid hash: %v", err)
	}

	opener := c.Opener
	if opener == nil {
		opener = &RealHashFileOpener{}
	}

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

	actualHash, err := c.computeHash(f, algorithm)
	if err != nil {
		return result.Failf("failed to compute hash: %v", err)
	}

	if actualHash != expectedHash {
		return result.Failf("hash mismatch\n       expected: %s\n       actual:   %s", expectedHash, actualHash)
	}

	result.Status = check.StatusOK
	result.AddDetailf("algorithm: %s", algorithm)
	result.AddDetailf("hash: %s", actualHash)
	return result
}

func (c *Check) validateHash(hashStr string, algorithm HashAlgorithm) error {
	if _, err := hex.DecodeString(hashStr); err != nil {
		return fmt.Errorf("not valid hexadecimal")
	}

	expectedLen := algorithm.ExpectedHexLength()
	if len(hashStr) != expectedLen {
		return fmt.Errorf("expected %d characters for %s, got %d", expectedLen, algorithm, len(hashStr))
	}

	return nil
}

func (c *Check) computeHash(r io.Reader, algorithm HashAlgorithm) (string, error) {
	h := algorithm.NewHasher()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var bsdFormatRegex = regexp.MustCompile(`^(SHA256|SHA384|SHA512|SHA1|MD5)\s+\((.+)\)\s*=\s*([a-fA-F0-9]+)$`)

func (c *Check) parseChecksumFile() (string, HashAlgorithm, error) {
	opener := c.Opener
	if opener == nil {
		opener = &RealHashFileOpener{}
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

		if hashVal, algo, ok := parseBSDLine(line, targetFile, c.File); ok {
			return hashVal, algo, nil
		}

		if hashVal, algo, ok := parseGNULine(line, targetFile, c.File); ok {
			return hashVal, algo, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("error reading checksum file: %v", err)
	}

	return "", "", fmt.Errorf("file %q not found in checksum file", targetFile)
}

func parseBSDLine(line, targetFile, fullPath string) (hashStr string, algo HashAlgorithm, ok bool) {
	matches := bsdFormatRegex.FindStringSubmatch(line)
	if matches == nil {
		return "", "", false
	}

	algoStr, filename, hashValue := matches[1], matches[2], matches[3]
	if filename != targetFile && filename != fullPath {
		return "", "", false
	}

	algo = AlgorithmSHA256
	switch algoStr {
	case "SHA384":
		algo = AlgorithmSHA384
	case "SHA512":
		algo = AlgorithmSHA512
	case "SHA1":
		algo = AlgorithmSHA1
	case "MD5":
		algo = AlgorithmMD5
	}

	return hashValue, algo, true
}

func parseGNULine(line, targetFile, fullPath string) (hashStr string, algo HashAlgorithm, ok bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", "", false
	}

	hashValue := parts[0]
	filename := strings.TrimPrefix(parts[len(parts)-1], "*")

	if filename != targetFile && filename != fullPath {
		return "", "", false
	}

	algo = AlgorithmSHA256
	switch len(hashValue) {
	case 128:
		algo = AlgorithmSHA512
	case 96:
		algo = AlgorithmSHA384
	case 40:
		algo = AlgorithmSHA1
	case 32:
		algo = AlgorithmMD5
	}

	return hashValue, algo, true
}
