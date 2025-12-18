package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/KilimcininKorOglu/poros/internal/output"
	"github.com/KilimcininKorOglu/poros/internal/trace"
	"github.com/KilimcininKorOglu/poros/internal/tui"
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
	config.DestPort = destPort

	// Configure enrichment
	config.EnableEnrichment = !noEnrich
	config.EnableRDNS = !noRDNS && !noEnrich
	config.EnableASN = !noASN && !noEnrich
	config.EnableGeoIP = !noGeoIP && !noEnrich

	// Set probe method
	if useParis {
		config.ProbeMethod = trace.ProbeParis
		config.Paris = true
	} else if useUDP {
		config.ProbeMethod = trace.ProbeUDP
	} else if useTCP {
		config.ProbeMethod = trace.ProbeTCP
	} else {
		config.ProbeMethod = trace.ProbeICMP
	}

	// If TUI mode requested, run TUI
	if tuiMode {
		return tui.Run(target, config)
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

	// Only show header for text output
	if !jsonOutput && !csvOutput {
		fmt.Fprintf(os.Stderr, "Tracing route to %s, %d hops max\n\n", target, maxHops)
	}

	result, err := tracer.Trace(ctx, target)
	if err != nil {
		return fmt.Errorf("trace failed: %w", err)
	}

	// Configure output
	outputConfig := output.Config{
		Colors:     !noColor,
		NoHostname: false,
		NoASN:      noASN,
		NoGeoIP:    noGeoIP,
	}

	// Determine output format
	var format output.Format
	switch {
	case jsonOutput:
		format = output.FormatJSON
	case csvOutput:
		format = output.FormatCSV
	case verbose:
		format = output.FormatVerbose
	default:
		format = output.FormatText
	}

	// Create writer and output results
	writer := output.NewWriter(format, outputConfig)
	if err := writer.Write(result); err != nil {
		return err
	}

	// Generate HTML report if requested
	if htmlOutput != "" {
		htmlFormatter := output.NewHTMLFormatter(outputConfig)
		if err := output.WriteToFile(result, htmlOutput, htmlFormatter); err != nil {
			return fmt.Errorf("failed to write HTML report: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nHTML report saved to: %s\n", htmlOutput)
	}

	return nil
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
