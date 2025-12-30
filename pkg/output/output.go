package output

import (
	"fmt"
	"os"
	"strings"

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
	var indent string
	if r.OK() {
		fmt.Printf("%s[OK]%s %s\n", green, reset, formatLabel(r.Name))
		indent = "     " // align with content after "[OK] "
		for _, d := range r.Details {
			fmt.Printf("%s%s\n", indent, formatLabel(d))
		}
	} else {
		fmt.Printf("%s[FAIL]%s %s\n", red, reset, formatLabel(r.Name))
		indent = "       " // align with content after "[FAIL] "
		for _, d := range r.Details {
			fmt.Printf("%s%s%s%s\n", indent, red, d, reset)
		}
	}
}

// formatLabel colors the label part (before colon) in dim.
func formatLabel(s string) string {
	if idx := strings.Index(s, ":"); idx != -1 {
		return dim + s[:idx+1] + reset + s[idx+1:]
	}
	return s
}
