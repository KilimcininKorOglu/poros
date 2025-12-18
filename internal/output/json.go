package output

import (
	"encoding/json"

	"github.com/KilimcininKorOglu/poros/internal/trace"
)

// JSONFormatter formats trace results as JSON.
type JSONFormatter struct {
	config Config
	pretty bool
}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter(config Config) *JSONFormatter {
	return &JSONFormatter{
		config: config,
		pretty: true, // Default to pretty-printed
	}
}

// NewJSONFormatterCompact creates a JSON formatter with compact output.
func NewJSONFormatterCompact(config Config) *JSONFormatter {
	return &JSONFormatter{
		config: config,
		pretty: false,
	}
}

// SetPretty enables or disables pretty-printing.
func (f *JSONFormatter) SetPretty(pretty bool) {
	f.pretty = pretty
}

// Format formats the trace result as JSON.
func (f *JSONFormatter) Format(result *trace.TraceResult) ([]byte, error) {
	// Convert to JSON-friendly output structure
	output := f.toJSONOutput(result)

	if f.pretty {
		return json.MarshalIndent(output, "", "  ")
	}
	return json.Marshal(output)
}

// JSONOutput is the JSON-serializable representation of a trace result.
type JSONOutput struct {
	Target      string      `json:"target"`
	ResolvedIP  string      `json:"resolved_ip"`
	Timestamp   string      `json:"timestamp"`
	ProbeMethod string      `json:"probe_method"`
	Completed   bool        `json:"completed"`
	Hops        []JSONHop   `json:"hops"`
	Summary     JSONSummary `json:"summary"`
}

// JSONHop represents a single hop in JSON format.
type JSONHop struct {
	Hop         int       `json:"hop"`
	IP          string    `json:"ip,omitempty"`
	Hostname    string    `json:"hostname,omitempty"`
	ASN         *JSONASN  `json:"asn,omitempty"`
	Geo         *JSONGeo  `json:"geo,omitempty"`
	RTTs        []float64 `json:"rtts"`
	AvgRTT      float64   `json:"avg_rtt_ms"`
	MinRTT      float64   `json:"min_rtt_ms"`
	MaxRTT      float64   `json:"max_rtt_ms"`
	Jitter      float64   `json:"jitter_ms"`
	LossPercent float64   `json:"loss_percent"`
	Responded   bool      `json:"responded"`
}

// JSONASN represents ASN information in JSON format.
type JSONASN struct {
	Number  int    `json:"number"`
	Org     string `json:"org"`
	Country string `json:"country,omitempty"`
}

// JSONGeo represents geographic information in JSON format.
type JSONGeo struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}

// JSONSummary represents trace summary in JSON format.
type JSONSummary struct {
	TotalHops         int     `json:"total_hops"`
	TotalTimeMs       float64 `json:"total_time_ms"`
	PacketLossPercent float64 `json:"packet_loss_percent"`
}

// toJSONOutput converts a TraceResult to JSONOutput.
func (f *JSONFormatter) toJSONOutput(result *trace.TraceResult) *JSONOutput {
	output := &JSONOutput{
		Target:      result.Target,
		ResolvedIP:  result.ResolvedIP.String(),
		Timestamp:   result.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		ProbeMethod: result.ProbeMethod,
		Completed:   result.Completed,
		Hops:        make([]JSONHop, len(result.Hops)),
		Summary: JSONSummary{
			TotalHops:         result.Summary.TotalHops,
			TotalTimeMs:       roundFloat(result.Summary.TotalTimeMs, 3),
			PacketLossPercent: roundFloat(result.Summary.PacketLossPercent, 1),
		},
	}

	for i, hop := range result.Hops {
		output.Hops[i] = f.toJSONHop(&hop)
	}

	return output
}

// toJSONHop converts a Hop to JSONHop.
func (f *JSONFormatter) toJSONHop(hop *trace.Hop) JSONHop {
	jh := JSONHop{
		Hop:         hop.Number,
		RTTs:        hop.RTTs,
		AvgRTT:      roundFloat(hop.AvgRTT, 3),
		MinRTT:      roundFloat(hop.MinRTT, 3),
		MaxRTT:      roundFloat(hop.MaxRTT, 3),
		Jitter:      roundFloat(hop.Jitter, 3),
		LossPercent: roundFloat(hop.LossPercent, 1),
		Responded:   hop.Responded,
	}

	if hop.IP != nil {
		jh.IP = hop.IP.String()
	}

	if hop.Hostname != "" {
		jh.Hostname = hop.Hostname
	}

	if hop.ASN != nil {
		jh.ASN = &JSONASN{
			Number:  hop.ASN.Number,
			Org:     hop.ASN.Org,
			Country: hop.ASN.Country,
		}
	}

	if hop.Geo != nil {
		jh.Geo = &JSONGeo{
			Country:     hop.Geo.Country,
			CountryCode: hop.Geo.CountryCode,
			City:        hop.Geo.City,
			Latitude:    hop.Geo.Latitude,
			Longitude:   hop.Geo.Longitude,
		}
	}

	return jh
}

// ContentType returns the MIME type for JSON output.
func (f *JSONFormatter) ContentType() string {
	return "application/json"
}

// FileExtension returns the file extension for JSON output.
func (f *JSONFormatter) FileExtension() string {
	return "json"
}

// Helper function to round floats
func roundFloat(val float64, precision int) float64 {
	if precision == 0 {
		return float64(int(val + 0.5))
	}
	p := float64(1)
	for i := 0; i < precision; i++ {
		p *= 10
	}
	return float64(int(val*p+0.5)) / p
}
