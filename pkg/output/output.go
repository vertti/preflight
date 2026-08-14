package output

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/term"

	"github.com/vertti/preflight/pkg/check"
)

var (
	green = "\033[32m"
	red   = "\033[31m"
	dim   = "\033[90m"
	reset = "\033[0m"
)

func init() {
	if !shouldEnableColor() {
		green, red, dim, reset = "", "", "", ""
	}
}

// shouldEnableColor determines if colors should be enabled.
// Priority: PREFLIGHT_COLOR > NO_COLOR > CLICOLOR_FORCE > CI detection > TTY detection
func shouldEnableColor() bool {
	// PREFLIGHT_COLOR=1 forces colors on (for Docker builds, etc.)
	if os.Getenv("PREFLIGHT_COLOR") == "1" {
		return true
	}
	// PREFLIGHT_COLOR=0 forces colors off
	if os.Getenv("PREFLIGHT_COLOR") == "0" {
		return false
	}

	// NO_COLOR standard (https://no-color.org/)
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}

	// CLICOLOR_FORCE=1 forces colors
	if os.Getenv("CLICOLOR_FORCE") == "1" {
		return true
	}

	// CI environment detection - enable colors in known CI systems
	if isCIEnvironment() {
		return true
	}

	// Fall back to TTY detection
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// isCIEnvironment detects common CI/CD platforms that support ANSI colors.
func isCIEnvironment() bool {
	// GitHub Actions - supports colors, sets GITHUB_ACTIONS=true
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return true
	}

	// GitLab CI - supports colors, sets GITLAB_CI=true
	if os.Getenv("GITLAB_CI") == "true" {
		return true
	}

	// CircleCI - supports colors, sets CIRCLECI=true
	if os.Getenv("CIRCLECI") == "true" {
		return true
	}

	// Travis CI - supports colors, sets TRAVIS=true
	if os.Getenv("TRAVIS") == "true" {
		return true
	}

	// Jenkins with AnsiColor plugin - sets TERM when colors are supported
	if os.Getenv("JENKINS_URL") != "" && os.Getenv("TERM") != "" {
		return true
	}

	// Azure Pipelines - sets TF_BUILD=True
	if os.Getenv("TF_BUILD") == "True" {
		return true
	}

	// Buildkite - supports colors, sets BUILDKITE=true
	if os.Getenv("BUILDKITE") == "true" {
		return true
	}

	// TeamCity - sets TEAMCITY_VERSION
	if os.Getenv("TEAMCITY_VERSION") != "" {
		return true
	}

	// Drone CI - sets DRONE=true
	if os.Getenv("DRONE") == "true" {
		return true
	}

	// Bitbucket Pipelines - sets BITBUCKET_BUILD_NUMBER
	if os.Getenv("BITBUCKET_BUILD_NUMBER") != "" {
		return true
	}

	// AWS CodeBuild - sets CODEBUILD_BUILD_ID
	if os.Getenv("CODEBUILD_BUILD_ID") != "" {
		return true
	}

	// Woodpecker CI - sets CI=woodpecker
	if os.Getenv("CI") == "woodpecker" {
		return true
	}

	return false
}

// PrintResult outputs a check result with colored status.
func PrintResult(r check.Result) {
	if r.OK() {
		fmt.Printf("%s[OK]%s %s\n", green, reset, formatLabel(sanitizeInline(r.Name)))
		// Align with content after "[OK] ".
		printDetails(r.Details, "     ", true)
		return
	}

	fmt.Printf("%s[FAIL]%s %s\n", red, reset, formatLabel(sanitizeInline(r.Name)))
	// Align with content after "[FAIL] ".
	printDetails(r.Details, "       ", false)
}

// printDetails writes each detail under the result line, indenting every line of
// it. A detail routinely spans several lines — a version banner, a stderr dump,
// an HTTP body — and indenting the continuation lines is what keeps a checked
// program from forging a result of its own: those start at column 0.
//
// Blank lines are left blank rather than indented, so no line carries trailing
// whitespace.
func printDetails(details []string, indent string, ok bool) {
	for _, d := range details {
		for i, line := range strings.Split(sanitizeBlock(d), "\n") {
			switch {
			case line == "":
				fmt.Println()
			case !ok:
				fmt.Printf("%s%s%s%s\n", indent, red, line, reset)
			case i == 0:
				// Only the opening line carries the "label:" that dimming applies to.
				fmt.Printf("%s%s\n", indent, formatLabel(line))
			default:
				fmt.Printf("%s%s\n", indent, line)
			}
		}
	}
}

// sanitizeInline renders every control character as an escape, newlines
// included. It is for text that has to stay on one line: a result's name shares
// its line with the [OK] marker, so a newline there would put whatever followed
// at column 0, which is exactly what a forged result line looks like.
func sanitizeInline(s string) string {
	return escapeControl(s, func(r rune) bool { return r == '\t' })
}

// sanitizeBlock keeps newlines for the caller to indent, and escapes every other
// control character. A carriage return still has to go: it returns to column 0
// and overwrites what is already there, so it could rewrite a real result line.
func sanitizeBlock(s string) string {
	return escapeControl(s, func(r rune) bool { return r == '\t' || r == '\n' })
}

// escapeControl renders control characters as escapes, except the ones keep
// accepts. Text a checked program controls — a version banner, an HTTP body, an
// environment variable — must not be able to redraw the terminal or write
// preflight's own claims about what was verified.
//
// Tabs are kept throughout: they only ever move right, so they cannot reach
// column 0, and escaping them mangled the tab-aligned help text programs print
// when a version flag is wrong.
func escapeControl(s string, keep func(rune) bool) string {
	// Program output almost always ends in a newline, and keeping that one would
	// add a blank line under every such detail.
	s = strings.TrimRight(s, " \t\r\n")

	escaped := func(r rune) bool { return unicode.IsControl(r) && !keep(r) }
	if !strings.ContainsFunc(s, escaped) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if escaped(r) {
			// QuoteRune yields the familiar Go forms: '\n', '\x1b', ''.
			b.WriteString(strings.Trim(strconv.QuoteRune(r), "'"))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// formatLabel colors the label part (before colon) in dim.
func formatLabel(s string) string {
	if idx := strings.Index(s, ":"); idx != -1 {
		return dim + s[:idx+1] + reset + s[idx+1:]
	}
	return s
}
