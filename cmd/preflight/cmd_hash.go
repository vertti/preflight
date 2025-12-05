package main

import (
	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/hashcheck"
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

func runHashCheck(_ *cobra.Command, args []string) error {
	file := args[0]

	// Validate exactly one hash flag is set
	if err := requireExactlyOne(
		flagValue{"--sha256", hashSHA256},
		flagValue{"--sha512", hashSHA512},
		flagValue{"--md5", hashMD5},
		flagValue{"--checksum-file", hashChecksumFile},
	); err != nil {
		return err
	}

	// Determine algorithm and expected hash from flags
	var algorithm hashcheck.HashAlgorithm
	var expectedHash string
	switch {
	case hashSHA256 != "":
		algorithm = hashcheck.AlgorithmSHA256
		expectedHash = hashSHA256
	case hashSHA512 != "":
		algorithm = hashcheck.AlgorithmSHA512
		expectedHash = hashSHA512
	case hashMD5 != "":
		algorithm = hashcheck.AlgorithmMD5
		expectedHash = hashMD5
	}

	c := &hashcheck.Check{
		File:         file,
		ExpectedHash: expectedHash,
		Algorithm:    algorithm,
		ChecksumFile: hashChecksumFile,
		Opener:       &hashcheck.RealFileOpener{},
	}

	return runCheck(c)
}
