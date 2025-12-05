package main

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/httpcheck"
	"github.com/vertti/preflight/pkg/output"
)

var (
	httpStatus     int
	httpTimeout    time.Duration
	httpMethod     string
	httpHeaders    []string
	httpInsecure   bool
	httpRetry      int
	httpRetryDelay time.Duration
)

var httpCmd = &cobra.Command{
	Use:   "http <url>",
	Short: "Check HTTP endpoint health",
	Args:  cobra.ExactArgs(1),
	RunE:  runHTTPCheck,
}

func init() {
	// Must-have flags
	httpCmd.Flags().IntVar(&httpStatus, "status", 200, "expected HTTP status code")
	httpCmd.Flags().DurationVar(&httpTimeout, "timeout", 5*time.Second, "request timeout")

	// Optional flags
	httpCmd.Flags().IntVar(&httpRetry, "retry", 0, "retry count on failure")
	httpCmd.Flags().DurationVar(&httpRetryDelay, "retry-delay", 1*time.Second, "delay between retries")
	httpCmd.Flags().StringVar(&httpMethod, "method", "GET", "HTTP method (GET or HEAD)")
	httpCmd.Flags().StringSliceVar(&httpHeaders, "header", nil, "custom header (key:value), can be repeated")
	httpCmd.Flags().BoolVar(&httpInsecure, "insecure", false, "skip TLS certificate verification")

	rootCmd.AddCommand(httpCmd)
}

func runHTTPCheck(cmd *cobra.Command, args []string) error {
	url := args[0]

	headers := parseHeaders(httpHeaders)

	c := &httpcheck.Check{
		URL:            url,
		ExpectedStatus: httpStatus,
		Timeout:        httpTimeout,
		Method:         httpMethod,
		Headers:        headers,
		Insecure:       httpInsecure,
		Retry:          httpRetry,
		RetryDelay:     httpRetryDelay,
		Client:         &httpcheck.RealHTTPClient{Timeout: httpTimeout, Insecure: httpInsecure},
	}

	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}

// parseHeaders converts ["key:value", ...] to map[string]string
func parseHeaders(headers []string) map[string]string {
	result := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}
