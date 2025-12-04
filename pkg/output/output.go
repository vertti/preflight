package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/jwalton/go-supportscolor"

	"github.com/vertti/preflight/pkg/check"
)

var (
	green = "\033[32m"
	red   = "\033[31m"
	dim   = "\033[90m"
	reset = "\033[0m"
)

func init() {
	colorEnabled := supportscolor.Stdout().SupportsColor

	// PREFLIGHT_COLOR=1 forces colors (useful for Docker builds)
	if os.Getenv("PREFLIGHT_COLOR") == "1" {
		colorEnabled = true
	}

	if !colorEnabled {
		green, red, dim, reset = "", "", "", ""
	}
}

// PrintResult outputs a check result with colored status.
func PrintResult(r check.Result) {
	var indent string
	if r.OK() {
		fmt.Printf("%s[OK]%s %s\n", green, reset, formatLabel(r.Name))
		indent = "     " // align with content after "[OK] "
	} else {
		fmt.Printf("%s[FAIL]%s %s\n", red, reset, formatLabel(r.Name))
		indent = "       " // align with content after "[FAIL] "
	}
	for _, d := range r.Details {
		fmt.Printf("%s%s\n", indent, formatLabel(d))
	}
}

// formatLabel colors the label part (before colon) in dim.
func formatLabel(s string) string {
	if idx := strings.Index(s, ":"); idx != -1 {
		return dim + s[:idx+1] + reset + s[idx+1:]
	}
	return s
}
