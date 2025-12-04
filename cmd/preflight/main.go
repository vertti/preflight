package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/cmdcheck"
	"github.com/vertti/preflight/pkg/envcheck"
	"github.com/vertti/preflight/pkg/filecheck"
	"github.com/vertti/preflight/pkg/preflightfile"
	"github.com/vertti/preflight/pkg/tcpcheck"
	"github.com/vertti/preflight/pkg/version"
)

// Version is set at build time via ldflags
var Version = "dev"

func main() {
	if len(os.Args) > 1 {
		firstArg := os.Args[1]

		if !strings.HasPrefix(firstArg, "-") {
			knownSubcommands := []string{"cmd", "env", "file", "tcp", "run", "version", "help", "--help", "-h"}
			isSubcommand := false
			for _, subcmd := range knownSubcommands {
				if firstArg == subcmd {
					isSubcommand = true
					break
				}
			}

			if !isSubcommand {
				if info, err := os.Stat(firstArg); err == nil && !info.IsDir() {
					runFile = firstArg
					os.Args = append([]string{os.Args[0], "run"}, os.Args[2:]...)
				}
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "preflight",
	Short:   "Docker preflight checks for your runtime environment",
	Long:    "Preflight is a CLI tool for running sanity checks on container and CI environments.",
	Version: Version,
}

var cmdCmd = &cobra.Command{
	Use:   "cmd <command>",
	Short: "Check that a command exists and can run",
	Args:  cobra.ExactArgs(1),
	RunE:  runCmdCheck,
}

var envCmd = &cobra.Command{
	Use:   "env <variable>",
	Short: "Check that an environment variable is set",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvCheck,
}

var fileCmd = &cobra.Command{
	Use:   "file <path>",
	Short: "Check that a file or directory exists and meets requirements",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileCheck,
}

var tcpCmd = &cobra.Command{
	Use:   "tcp <host:port>",
	Short: "Check TCP connectivity to a host:port",
	Args:  cobra.ExactArgs(1),
	RunE:  runTCPCheck,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run checks from a .preflight file",
	Args:  cobra.NoArgs,
	RunE:  runRun,
}

var (
	// cmd flags
	minVersion   string
	maxVersion   string
	exactVersion string
	matchPattern string
	versionCmd   string

	// env flags
	envRequired  bool
	envMatch     string
	envExact     string
	envOneOf     []string
	envHideValue bool
	envMaskValue bool

	// file flags
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

	// tcp flags
	tcpTimeout time.Duration

	// run flags
	runFile string
)

func init() {
	// cmd subcommand
	cmdCmd.Flags().StringVar(&minVersion, "min", "", "minimum version required (inclusive)")
	cmdCmd.Flags().StringVar(&maxVersion, "max", "", "maximum version allowed (exclusive)")
	cmdCmd.Flags().StringVar(&exactVersion, "exact", "", "exact version required")
	cmdCmd.Flags().StringVar(&matchPattern, "match", "", "regex pattern to match against version output")
	cmdCmd.Flags().StringVar(&versionCmd, "version-cmd", "--version", "command to get version")
	rootCmd.AddCommand(cmdCmd)

	// env subcommand
	envCmd.Flags().BoolVar(&envRequired, "required", false, "fail if not set (allows empty)")
	envCmd.Flags().StringVar(&envMatch, "match", "", "regex pattern to match value")
	envCmd.Flags().StringVar(&envExact, "exact", "", "exact value required")
	envCmd.Flags().StringSliceVar(&envOneOf, "one-of", nil, "value must be one of these (comma-separated)")
	envCmd.Flags().BoolVar(&envHideValue, "hide-value", false, "don't show value in output")
	envCmd.Flags().BoolVar(&envMaskValue, "mask-value", false, "show masked value (first/last 3 chars)")
	rootCmd.AddCommand(envCmd)

	// file subcommand
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

	// tcp subcommand
	tcpCmd.Flags().DurationVar(&tcpTimeout, "timeout", 5*time.Second, "connection timeout")
	rootCmd.AddCommand(tcpCmd)

	// run subcommand
	runCmd.Flags().StringVar(&runFile, "file", "", "path to .preflight file (default: search up from current directory)")
	rootCmd.AddCommand(runCmd)
}

func runCmdCheck(cmd *cobra.Command, args []string) error {
	commandName := args[0]

	c := &cmdcheck.Check{
		Name:         commandName,
		VersionArgs:  parseVersionArgs(versionCmd),
		MatchPattern: matchPattern,
		Runner:       &cmdcheck.RealRunner{},
	}

	if minVersion != "" {
		v, err := version.Parse(minVersion)
		if err != nil {
			return fmt.Errorf("invalid --min version: %w", err)
		}
		c.MinVersion = &v
	}

	if maxVersion != "" {
		v, err := version.Parse(maxVersion)
		if err != nil {
			return fmt.Errorf("invalid --max version: %w", err)
		}
		c.MaxVersion = &v
	}

	if exactVersion != "" {
		v, err := version.Parse(exactVersion)
		if err != nil {
			return fmt.Errorf("invalid --exact version: %w", err)
		}
		c.ExactVersion = &v
	}

	result := c.Run()
	printResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}

func parseVersionArgs(s string) []string {
	return strings.Fields(s)
}

func runEnvCheck(cmd *cobra.Command, args []string) error {
	varName := args[0]

	c := &envcheck.Check{
		Name:      varName,
		Required:  envRequired,
		Match:     envMatch,
		Exact:     envExact,
		OneOf:     envOneOf,
		HideValue: envHideValue,
		MaskValue: envMaskValue,
		Getter:    &envcheck.RealEnvGetter{},
	}

	result := c.Run()
	printResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}

func runFileCheck(cmd *cobra.Command, args []string) error {
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

	result := c.Run()
	printResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}

func runTCPCheck(cmd *cobra.Command, args []string) error {
	address := args[0]

	c := &tcpcheck.Check{
		Address: address,
		Timeout: tcpTimeout,
		Dialer:  &tcpcheck.RealDialer{},
	}

	result := c.Run()
	printResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}

func runRun(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	preflightPath, err := preflightfile.FindFile(wd, runFile)
	if err != nil {
		return err
	}

	commands, err := preflightfile.ParseFile(preflightPath)
	if err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	for _, command := range commands {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			continue
		}

		if parts[0] == "preflight" {
			parts[0] = executable
		}

		execCmd := exec.Command(parts[0], parts[1:]...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				os.Exit(exitError.ExitCode())
			}
			return fmt.Errorf("failed to execute command %q: %w", command, err)
		}
	}

	return nil
}

func printResult(r check.Result) {
	if r.OK() {
		fmt.Printf("[OK] %s\n", r.Name)
	} else {
		fmt.Printf("[FAIL] %s\n", r.Name)
	}
	for _, d := range r.Details {
		fmt.Printf("      %s\n", d)
	}
}
