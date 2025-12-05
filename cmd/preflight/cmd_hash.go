package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/hashcheck"
	"github.com/vertti/preflight/pkg/output"
)

var (
	hashSHA256       string
	hashSHA512       string
	hashMD5          string
	hashChecksumFile string
)

var hashCmd = &cobra.Command{
	Use:   "hash <file>",
	Short: "Verify file checksum",
	Long: `Verify a file's checksum against expected SHA256, SHA512, or MD5 hash.

Examples:
  preflight hash --sha256 67574ee...2cf myfile.tar.gz
  preflight hash --sha512 abc123... /path/to/file
  preflight hash --checksum-file SHASUMS256.txt node-v20.tar.gz`,
	Args: cobra.ExactArgs(1),
	RunE: runHashCheck,
}

func init() {
	hashCmd.Flags().StringVar(&hashSHA256, "sha256", "", "expected SHA256 hash")
	hashCmd.Flags().StringVar(&hashSHA512, "sha512", "", "expected SHA512 hash")
	hashCmd.Flags().StringVar(&hashMD5, "md5", "", "expected MD5 hash (legacy)")
	hashCmd.Flags().StringVar(&hashChecksumFile, "checksum-file", "", "verify against checksum file (GNU or BSD format)")

	rootCmd.AddCommand(hashCmd)
}

func runHashCheck(cmd *cobra.Command, args []string) error {
	file := args[0]

	// Determine algorithm and expected hash from flags
	var algorithm hashcheck.HashAlgorithm
	var expectedHash string

	flagCount := 0
	if hashSHA256 != "" {
		algorithm = hashcheck.AlgorithmSHA256
		expectedHash = hashSHA256
		flagCount++
	}
	if hashSHA512 != "" {
		algorithm = hashcheck.AlgorithmSHA512
		expectedHash = hashSHA512
		flagCount++
	}
	if hashMD5 != "" {
		algorithm = hashcheck.AlgorithmMD5
		expectedHash = hashMD5
		flagCount++
	}
	if hashChecksumFile != "" {
		flagCount++
	}

	if flagCount == 0 {
		return fmt.Errorf("one of --sha256, --sha512, --md5, or --checksum-file is required")
	}
	if flagCount > 1 {
		return fmt.Errorf("only one of --sha256, --sha512, --md5, or --checksum-file can be specified")
	}

	c := &hashcheck.Check{
		File:         file,
		ExpectedHash: expectedHash,
		Algorithm:    algorithm,
		ChecksumFile: hashChecksumFile,
		Opener:       &hashcheck.RealFileOpener{},
	}

	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}
