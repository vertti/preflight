package main

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/promcheck"
)

var (
	promQuery      string
	promMin        float64
	promMax        float64
	promExact      float64
	promMinSet     bool
	promMaxSet     bool
	promExactSet   bool
	promTimeout    time.Duration
	promRetry      int
	promRetryDelay time.Duration
	promInsecure   bool
	promHeaders    []string
)

var prometheusCmd = &cobra.Command{
	Use:   "prometheus <url>",
	Short: "Check Prometheus metric value",
	Long: `Query a Prometheus server and validate metric values against thresholds.

Examples:
  preflight prometheus http://prometheus:9090 --query 'up{job="myapp"}' --exact 1
  preflight prometheus http://prometheus:9090 --query 'error_rate' --max 0.05
  preflight prometheus http://prometheus:9090 --query 'request_count' --min 100`,
	Args: cobra.ExactArgs(1),
	RunE: runPrometheusCheck,
}

func init() {
	prometheusCmd.Flags().StringVar(&promQuery, "query", "", "PromQL query (required)")
	_ = prometheusCmd.MarkFlagRequired("query")

	prometheusCmd.Flags().Float64Var(&promMin, "min", 0, "minimum value (fail if below)")
	prometheusCmd.Flags().Float64Var(&promMax, "max", 0, "maximum value (fail if above)")
	prometheusCmd.Flags().Float64Var(&promExact, "exact", 0, "exact value match")

	prometheusCmd.Flags().DurationVar(&promTimeout, "timeout", 5*time.Second, "request timeout")
	prometheusCmd.Flags().IntVar(&promRetry, "retry", 0, "retry count on failure")
	prometheusCmd.Flags().DurationVar(&promRetryDelay, "retry-delay", 1*time.Second, "delay between retries")
	prometheusCmd.Flags().BoolVar(&promInsecure, "insecure", false, "skip TLS certificate verification")
	prometheusCmd.Flags().StringSliceVar(&promHeaders, "header", nil, "custom header (key:value), can be repeated")

	rootCmd.AddCommand(prometheusCmd)
}

func runPrometheusCheck(cmd *cobra.Command, args []string) error {
	url := args[0]

	// Track which flags were explicitly set
	promMinSet = cmd.Flags().Changed("min")
	promMaxSet = cmd.Flags().Changed("max")
	promExactSet = cmd.Flags().Changed("exact")

	headers := parsePrometheusHeaders(promHeaders)

	c := &promcheck.Check{
		URL:        url,
		Query:      promQuery,
		Timeout:    promTimeout,
		Retry:      promRetry,
		RetryDelay: promRetryDelay,
		Insecure:   promInsecure,
		Headers:    headers,
		Client:     &promcheck.RealHTTPClient{Timeout: promTimeout, Insecure: promInsecure},
	}

	// Only set threshold pointers if flags were explicitly provided
	if promMinSet {
		c.Min = &promMin
	}
	if promMaxSet {
		c.Max = &promMax
	}
	if promExactSet {
		c.Exact = &promExact
	}

	return runCheck(c)
}

// parsePrometheusHeaders converts ["key:value", ...] to map[string]string
func parsePrometheusHeaders(headers []string) map[string]string {
	result := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}
