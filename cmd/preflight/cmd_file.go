package main

import (
	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/filecheck"
)

var (
	fileDir        bool
	fileWritable   bool
	fileExecutable bool
	fileNotEmpty   bool
	fileMinSize    int64
	fileMaxSize    int64
	fileMatch      string
	fileContains   string
	fileHead       int64
	fileMode       string
	fileModeExact  string
)

var fileCmd = &cobra.Command{
	Use:   "file <path>",
	Short: "Check that a file or directory exists and meets requirements",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileCheck,
}

func init() {
	fileCmd.Flags().BoolVar(&fileDir, "dir", false, "expect a directory")
	fileCmd.Flags().BoolVar(&fileWritable, "writable", false, "check write permission")
	fileCmd.Flags().BoolVar(&fileExecutable, "executable", false, "check execute permission")
	fileCmd.Flags().BoolVar(&fileNotEmpty, "not-empty", false, "file must have size > 0")
	fileCmd.Flags().Int64Var(&fileMinSize, "min-size", 0, "minimum file size in bytes")
	fileCmd.Flags().Int64Var(&fileMaxSize, "max-size", 0, "maximum file size in bytes")
	fileCmd.Flags().StringVar(&fileMatch, "match", "", "regex pattern to match content")
	fileCmd.Flags().StringVar(&fileContains, "contains", "", "literal string to search in content")
	fileCmd.Flags().Int64Var(&fileHead, "head", 0, "limit content read to first N bytes")
	fileCmd.Flags().StringVar(&fileMode, "mode", "", "minimum permissions (e.g., 0644)")
	fileCmd.Flags().StringVar(&fileModeExact, "mode-exact", "", "exact permissions required")
	rootCmd.AddCommand(fileCmd)
}

func runFileCheck(_ *cobra.Command, args []string) error {
	path := args[0]

	c := &filecheck.Check{
		Path:       path,
		ExpectDir:  fileDir,
		Writable:   fileWritable,
		Executable: fileExecutable,
		NotEmpty:   fileNotEmpty,
		MinSize:    fileMinSize,
		MaxSize:    fileMaxSize,
		Match:      fileMatch,
		Contains:   fileContains,
		Head:       fileHead,
		Mode:       fileMode,
		ModeExact:  fileModeExact,
		FS:         &filecheck.RealFileSystem{},
	}

	return runCheck(c)
}
