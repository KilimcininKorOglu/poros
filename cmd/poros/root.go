package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/KilimcininKorOglu/poros/internal/config"
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

	// Config file
	cfgFile string
	cfg     *config.Config
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
  • Configuration file support (~/.config/poros/config.yaml)

Examples:
  poros google.com              Basic trace using ICMP
  poros -U google.com           Use UDP probes
  poros -T --port 443 host      TCP probe to port 443
  poros -v google.com           Verbose table output
  poros --json google.com       JSON output
  poros --tui google.com        Interactive TUI mode
  poros config --init           Create default config file`,
	Args:              cobra.ExactArgs(1),
	PersistentPreRunE: loadConfig,
	RunE:              runTrace,
}

func init() {
	// Config file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default: ~/.config/poros/config.yaml)")

	// Probe method flags
	rootCmd.Flags().BoolVarP(&useICMP, "icmp", "I", false, "Use ICMP Echo probes (default)")
	rootCmd.Flags().BoolVarP(&useUDP, "udp", "U", false, "Use UDP probes")
	rootCmd.Flags().BoolVarP(&useTCP, "tcp", "T", false, "Use TCP SYN probes")
	rootCmd.Flags().BoolVar(&useParis, "paris", false, "Use Paris traceroute algorithm")

	// Trace parameters
	rootCmd.Flags().IntVarP(&maxHops, "max-hops", "m", 0, "Maximum number of hops")
	rootCmd.Flags().IntVarP(&probeCount, "queries", "q", 0, "Number of probes per hop")
	rootCmd.Flags().DurationVarP(&timeout, "timeout", "w", 0, "Probe timeout")
	rootCmd.Flags().IntVarP(&firstHop, "first-hop", "f", 0, "Start from specified hop")
	rootCmd.Flags().BoolVar(&sequential, "sequential", false, "Use sequential mode (slower but reliable)")

	// Network settings
	rootCmd.Flags().BoolVarP(&forceIPv4, "ipv4", "4", false, "Use IPv4 only")
	rootCmd.Flags().BoolVarP(&forceIPv6, "ipv6", "6", false, "Use IPv6 only")
	rootCmd.Flags().StringVarP(&ifaceName, "interface", "i", "", "Network interface to use")
	rootCmd.Flags().StringVarP(&sourceIP, "source", "s", "", "Source IP address")
	rootCmd.Flags().IntVarP(&destPort, "port", "p", 0, "Destination port (UDP/TCP)")

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

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
}

// loadConfig loads configuration from file and applies defaults
func loadConfig(cmd *cobra.Command, args []string) error {
	var err error

	if cfgFile != "" {
		cfg, err = config.LoadFrom(cfgFile)
	} else {
		cfg, err = config.Load()
	}

	if err != nil {
		// If config file specified but not found, error
		if cfgFile != "" {
			return fmt.Errorf("failed to load config: %w", err)
		}
		// Otherwise use defaults
		cfg = config.DefaultConfig()
	}

	// Apply config defaults if flags not explicitly set
	applyConfigDefaults(cmd)

	return nil
}

// applyConfigDefaults applies config file values for unset flags
func applyConfigDefaults(cmd *cobra.Command) {
	if cfg == nil {
		return
	}

	defaults := cfg.Defaults

	// Output mode from config (if no flag set)
	if !cmd.Flags().Changed("tui") && defaults.TUI {
		tuiMode = true
	}
	if !cmd.Flags().Changed("verbose") && defaults.Verbose {
		verbose = true
	}
	if !cmd.Flags().Changed("json") && defaults.JSON {
		jsonOutput = true
	}
	if !cmd.Flags().Changed("csv") && defaults.CSV {
		csvOutput = true
	}
	if !cmd.Flags().Changed("no-color") && defaults.NoColor {
		noColor = true
	}

	// Probe method from config
	if !cmd.Flags().Changed("paris") && defaults.Paris {
		useParis = true
	}
	if !cmd.Flags().Changed("icmp") && !cmd.Flags().Changed("udp") && !cmd.Flags().Changed("tcp") {
		switch defaults.ProbeMethod {
		case "udp":
			useUDP = true
		case "tcp":
			useTCP = true
		}
	}

	// Trace parameters from config
	if !cmd.Flags().Changed("max-hops") {
		if defaults.MaxHops > 0 {
			maxHops = defaults.MaxHops
		} else {
			maxHops = 30
		}
	}
	if !cmd.Flags().Changed("queries") {
		if defaults.Queries > 0 {
			probeCount = defaults.Queries
		} else {
			probeCount = 3
		}
	}
	if !cmd.Flags().Changed("timeout") {
		if defaults.Timeout > 0 {
			timeout = defaults.Timeout
		} else {
			timeout = 3 * time.Second
		}
	}
	if !cmd.Flags().Changed("first-hop") {
		if defaults.FirstHop > 0 {
			firstHop = defaults.FirstHop
		} else {
			firstHop = 1
		}
	}
	if !cmd.Flags().Changed("sequential") && defaults.Sequential {
		sequential = true
	}

	// Network settings from config
	if !cmd.Flags().Changed("ipv4") && defaults.IPv4 {
		forceIPv4 = true
	}
	if !cmd.Flags().Changed("ipv6") && defaults.IPv6 {
		forceIPv6 = true
	}
	if !cmd.Flags().Changed("port") {
		if defaults.Port > 0 {
			destPort = defaults.Port
		} else {
			destPort = 33434
		}
	}

	// Enrichment from config
	if !defaults.Enrichment.Enabled {
		noEnrich = true
	}
	if !cmd.Flags().Changed("no-rdns") && !defaults.Enrichment.RDNS {
		noRDNS = true
	}
	if !cmd.Flags().Changed("no-asn") && !defaults.Enrichment.ASN {
		noASN = true
	}
	if !cmd.Flags().Changed("no-geoip") && !defaults.Enrichment.GeoIP {
		noGeoIP = true
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Poros %s\n", version)
		fmt.Printf("  Commit: %s\n", commit)
		fmt.Printf("  Built:  %s\n", date)
		fmt.Printf("  Config: %s\n", config.GetConfigPath())
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `Manage Poros configuration file.

Commands:
  poros config --init     Create default config file
  poros config --show     Show current configuration
  poros config --path     Show config file path`,
	RunE: runConfig,
}

var (
	configInit bool
	configShow bool
	configPath bool
)

func init() {
	configCmd.Flags().BoolVar(&configInit, "init", false, "Create default config file")
	configCmd.Flags().BoolVar(&configShow, "show", false, "Show current configuration")
	configCmd.Flags().BoolVar(&configPath, "path", false, "Show config file path")
}

func runConfig(cmd *cobra.Command, args []string) error {
	if configPath {
		fmt.Println(config.GetConfigPath())
		return nil
	}

	if configInit {
		path := config.GetConfigPath()
		
		// Check if file already exists
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config file already exists: %s", path)
		}

		// Create default config
		cfg := config.DefaultConfig()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}

		fmt.Printf("Created config file: %s\n", path)
		fmt.Println("\nEdit this file to customize defaults.")
		fmt.Println("Example: Set 'tui: true' under 'defaults:' to always use TUI mode.")
		return nil
	}

	if configShow {
		fmt.Println(config.GenerateExample())
		return nil
	}

	// No flag specified, show help
	return cmd.Help()
}

func runTrace(cmd *cobra.Command, args []string) error {
	target := args[0]

	// Check for aliases
	if cfg != nil && cfg.Aliases != nil {
		if alias, ok := cfg.Aliases[target]; ok {
			target = alias
		}
	}

	// Build tracer configuration
	traceConfig := trace.DefaultConfig()
	traceConfig.MaxHops = maxHops
	traceConfig.ProbeCount = probeCount
	traceConfig.Timeout = timeout
	traceConfig.FirstHop = firstHop
	traceConfig.Sequential = sequential
	traceConfig.IPv4 = forceIPv4
	traceConfig.IPv6 = forceIPv6
	traceConfig.DestPort = destPort

	// Configure enrichment
	traceConfig.EnableEnrichment = !noEnrich
	traceConfig.EnableRDNS = !noRDNS && !noEnrich
	traceConfig.EnableASN = !noASN && !noEnrich
	traceConfig.EnableGeoIP = !noGeoIP && !noEnrich

	// Set probe method
	if useParis {
		traceConfig.ProbeMethod = trace.ProbeParis
		traceConfig.Paris = true
	} else if useUDP {
		traceConfig.ProbeMethod = trace.ProbeUDP
	} else if useTCP {
		traceConfig.ProbeMethod = trace.ProbeTCP
	} else {
		traceConfig.ProbeMethod = trace.ProbeICMP
	}

	// If TUI mode requested, run TUI
	if tuiMode {
		return tui.Run(target, traceConfig)
	}

	// Create tracer
	tracer, err := trace.New(traceConfig)
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
