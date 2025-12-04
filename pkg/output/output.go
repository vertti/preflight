package output

import (
	"fmt"
	"os"

	"github.com/jwalton/go-supportscolor"

	"github.com/vertti/preflight/pkg/check"
)

var (
	green = "\033[32m"
	red   = "\033[31m"
	reset = "\033[0m"
)

func init() {
	colorEnabled := supportscolor.Stdout().SupportsColor

	// PREFLIGHT_COLOR=1 forces colors (useful for Docker builds)
	if os.Getenv("PREFLIGHT_COLOR") == "1" {
		colorEnabled = true
	}

	if !colorEnabled {
		green, red, reset = "", "", ""
	}
}

// PrintResult outputs a check result with colored status.
func PrintResult(r check.Result) {
	if r.OK() {
		fmt.Printf("%s[OK]%s %s\n", green, reset, r.Name)
	} else {
		fmt.Printf("%s[FAIL]%s %s\n", red, reset, r.Name)
	}
	for _, d := range r.Details {
		fmt.Printf("      %s\n", d)
	}
}
