package main

import (
	"context"
	"fmt"
	"time"

	"github.com/KilimcininKorOglu/poros/internal/trace"
	"github.com/spf13/cobra"
)

var (
	// Flags
	useICMP     bool
	useUDP      bool
	useTCP      bool
	useParis    bool
	maxHops     int
	probeCount  int
	timeout     time.Duration
	firstHop    int
	sequential  bool
	forceIPv4   bool
	forceIPv6   bool
	ifaceName   string
	sourceIP    string
	destPort    int
	verbose     bool
	jsonOutput  bool
	csvOutput   bool
	htmlOutput  string
	tuiMode     bool
	noEnrich    bool
	noRDNS      bool
	noASN       bool
	noGeoIP     bool
	noColor     bool
)

var rootCmd = &cobra.Command{
	Use:   "poros [flags] <target>",
	Short: "Modern network path tracer",
	Long: `Poros (Πόρος) - A modern, cross-platform network path tracer

Poros traces the route packets take to reach a destination host,
showing each hop along the path with timing information, ASN data,
and geographic location.

Features:
  • Multiple probe methods: ICMP (default), UDP, TCP SYN, Paris
  • Concurrent probing for fast results
  • ASN and GeoIP enrichment
  • Interactive TUI mode
  • Multiple output formats: text, JSON, CSV, HTML

Examples:
  poros google.com              Basic trace using ICMP
  poros -U google.com           Use UDP probes
  poros -T --port 443 host      TCP probe to port 443
  poros -v google.com           Verbose table output
  poros --json google.com       JSON output
  poros --tui google.com        Interactive TUI mode`,
	Args: cobra.ExactArgs(1),
	RunE: runTrace,
}

func init() {
	// Probe method flags
	rootCmd.Flags().BoolVarP(&useICMP, "icmp", "I", false, "Use ICMP Echo probes (default)")
	rootCmd.Flags().BoolVarP(&useUDP, "udp", "U", false, "Use UDP probes")
	rootCmd.Flags().BoolVarP(&useTCP, "tcp", "T", false, "Use TCP SYN probes")
	rootCmd.Flags().BoolVar(&useParis, "paris", false, "Use Paris traceroute algorithm")

	// Trace parameters
	rootCmd.Flags().IntVarP(&maxHops, "max-hops", "m", 30, "Maximum number of hops")
	rootCmd.Flags().IntVarP(&probeCount, "queries", "q", 3, "Number of probes per hop")
	rootCmd.Flags().DurationVarP(&timeout, "timeout", "w", 3*time.Second, "Probe timeout")
	rootCmd.Flags().IntVarP(&firstHop, "first-hop", "f", 1, "Start from specified hop")
	rootCmd.Flags().BoolVar(&sequential, "sequential", false, "Use sequential mode (slower but reliable)")

	// Network settings
	rootCmd.Flags().BoolVarP(&forceIPv4, "ipv4", "4", false, "Use IPv4 only")
	rootCmd.Flags().BoolVarP(&forceIPv6, "ipv6", "6", false, "Use IPv6 only")
	rootCmd.Flags().StringVarP(&ifaceName, "interface", "i", "", "Network interface to use")
	rootCmd.Flags().StringVarP(&sourceIP, "source", "s", "", "Source IP address")
	rootCmd.Flags().IntVarP(&destPort, "port", "p", 33434, "Destination port (UDP/TCP)")

	// Output flags
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed table output")
	rootCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
	rootCmd.Flags().BoolVar(&csvOutput, "csv", false, "Output in CSV format")
	rootCmd.Flags().StringVar(&htmlOutput, "html", "", "Generate HTML report to file")
	rootCmd.Flags().BoolVarP(&tuiMode, "tui", "t", false, "Interactive TUI mode")
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Enrichment flags
	rootCmd.Flags().BoolVar(&noEnrich, "no-enrich", false, "Disable all enrichment")
	rootCmd.Flags().BoolVar(&noRDNS, "no-rdns", false, "Disable reverse DNS lookups")
	rootCmd.Flags().BoolVar(&noASN, "no-asn", false, "Disable ASN lookups")
	rootCmd.Flags().BoolVar(&noGeoIP, "no-geoip", false, "Disable GeoIP lookups")

	// Version command
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Poros %s\n", version)
		fmt.Printf("  Commit: %s\n", commit)
		fmt.Printf("  Built:  %s\n", date)
	},
}

func runTrace(cmd *cobra.Command, args []string) error {
	target := args[0]

	// Build tracer configuration
	config := trace.DefaultConfig()
	config.MaxHops = maxHops
	config.ProbeCount = probeCount
	config.Timeout = timeout
	config.FirstHop = firstHop
	config.Sequential = sequential
	config.IPv4 = forceIPv4
	config.IPv6 = forceIPv6

	// Set probe method
	if useUDP {
		config.ProbeMethod = trace.ProbeUDP
	} else if useTCP {
		config.ProbeMethod = trace.ProbeTCP
	} else {
		config.ProbeMethod = trace.ProbeICMP
	}

	// Create tracer
	tracer, err := trace.New(config)
	if err != nil {
		return fmt.Errorf("failed to create tracer: %w", err)
	}
	defer tracer.Close()

	// Run trace
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	fmt.Printf("Tracing route to %s, %d hops max, %d byte packets\n\n", target, maxHops, 64)

	result, err := tracer.Trace(ctx, target)
	if err != nil {
		return fmt.Errorf("trace failed: %w", err)
	}

	// Print results
	printTraceResult(result)

	return nil
}

func printTraceResult(result *trace.TraceResult) {
	for _, hop := range result.Hops {
		fmt.Printf("%3d  ", hop.Number)

		if !hop.Responded {
			fmt.Println("* * *")
			continue
		}

		// Print IP (and hostname if available)
		if hop.Hostname != "" {
			fmt.Printf("%s (%s)  ", hop.Hostname, hop.IP)
		} else {
			fmt.Printf("%s  ", hop.IP)
		}

		// Print RTTs
		for _, rtt := range hop.RTTs {
			if rtt < 0 {
				fmt.Print("*  ")
			} else {
				fmt.Printf("%.3f ms  ", rtt)
			}
		}

		fmt.Println()
	}

	// Print summary
	fmt.Println()
	if result.Completed {
		fmt.Printf("Trace complete. %d hops, %.2f ms total\n", 
			result.Summary.TotalHops, result.Summary.TotalTimeMs)
	} else {
		fmt.Printf("Trace incomplete after %d hops\n", result.Summary.TotalHops)
	}
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets version information for the CLI.
func SetVersion(v, c, d string) {
	version = v
	commit = c
	date = d
}
