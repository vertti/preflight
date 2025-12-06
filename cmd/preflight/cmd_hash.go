package main

import (
	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/hashcheck"
)

var (
	hashSHA256       string
	hashSHA384       string
	hashSHA512       string
	hashSHA1         string
	hashMD5          string
	hashBLAKE2b      string
	hashBLAKE3       string
	hashAuto         string
	hashChecksumFile string
)

var hashCmd = &cobra.Command{
	Use:   "hash <file>",
	Short: "Verify file checksum",
	Long: `Verify a file's checksum against expected hash.

Supported algorithms: SHA256, SHA384, SHA512, SHA1, MD5, BLAKE2b-256, BLAKE3

Examples:
  preflight hash --sha256 67574ee...2cf myfile.tar.gz
  preflight hash --sha512 abc123... /path/to/file
  preflight hash --blake3 abc123... /path/to/file    # modern, fast
  preflight hash --auto abc123... /path/to/file      # auto-detect by length
  preflight hash --sha1 da39a3e...b3e /path/to/file  # legacy, use with caution
  preflight hash --checksum-file SHASUMS256.txt node-v20.tar.gz`,
	Args: cobra.ExactArgs(1),
	RunE: runHashCheck,
}

func init() {
	hashCmd.Flags().StringVar(&hashSHA256, "sha256", "", "expected SHA256 hash")
	hashCmd.Flags().StringVar(&hashSHA384, "sha384", "", "expected SHA384 hash")
	hashCmd.Flags().StringVar(&hashSHA512, "sha512", "", "expected SHA512 hash")
	hashCmd.Flags().StringVar(&hashSHA1, "sha1", "", "expected SHA1 hash (legacy, weak)")
	hashCmd.Flags().StringVar(&hashMD5, "md5", "", "expected MD5 hash (legacy, weak)")
	hashCmd.Flags().StringVar(&hashBLAKE2b, "blake2b-256", "", "expected BLAKE2b-256 hash")
	hashCmd.Flags().StringVar(&hashBLAKE3, "blake3", "", "expected BLAKE3 hash")
	hashCmd.Flags().StringVar(&hashAuto, "auto", "", "expected hash (auto-detect algorithm by length)")
	hashCmd.Flags().StringVar(&hashChecksumFile, "checksum-file", "", "verify against checksum file (GNU or BSD format)")

	rootCmd.AddCommand(hashCmd)
}

func runHashCheck(_ *cobra.Command, args []string) error {
	file := args[0]

	// Validate exactly one hash flag is set
	if err := requireExactlyOne(
		flagValue{"--sha256", hashSHA256},
		flagValue{"--sha384", hashSHA384},
		flagValue{"--sha512", hashSHA512},
		flagValue{"--sha1", hashSHA1},
		flagValue{"--md5", hashMD5},
		flagValue{"--blake2b-256", hashBLAKE2b},
		flagValue{"--blake3", hashBLAKE3},
		flagValue{"--auto", hashAuto},
		flagValue{"--checksum-file", hashChecksumFile},
	); err != nil {
		return err
	}

	// Determine algorithm and expected hash from flags
	var algorithm hashcheck.HashAlgorithm
	var expectedHash string
	var autoDetect bool
	switch {
	case hashSHA256 != "":
		algorithm = hashcheck.AlgorithmSHA256
		expectedHash = hashSHA256
	case hashSHA384 != "":
		algorithm = hashcheck.AlgorithmSHA384
		expectedHash = hashSHA384
	case hashSHA512 != "":
		algorithm = hashcheck.AlgorithmSHA512
		expectedHash = hashSHA512
	case hashSHA1 != "":
		algorithm = hashcheck.AlgorithmSHA1
		expectedHash = hashSHA1
	case hashMD5 != "":
		algorithm = hashcheck.AlgorithmMD5
		expectedHash = hashMD5
	case hashBLAKE2b != "":
		algorithm = hashcheck.AlgorithmBLAKE2b
		expectedHash = hashBLAKE2b
	case hashBLAKE3 != "":
		algorithm = hashcheck.AlgorithmBLAKE3
		expectedHash = hashBLAKE3
	case hashAuto != "":
		autoDetect = true
		expectedHash = hashAuto
	}

	c := &hashcheck.Check{
		File:         file,
		ExpectedHash: expectedHash,
		Algorithm:    algorithm,
		AutoDetect:   autoDetect,
		ChecksumFile: hashChecksumFile,
		Opener:       &hashcheck.RealHashFileOpener{},
	}

	return runCheck(c)
}
