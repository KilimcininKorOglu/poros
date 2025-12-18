package output

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/KilimcininKorOglu/poros/internal/trace"
)

// HTMLFormatter formats trace results as an HTML report.
type HTMLFormatter struct {
	config   Config
	template *template.Template
}

// NewHTMLFormatter creates a new HTML formatter.
func NewHTMLFormatter(config Config) *HTMLFormatter {
	tmpl := template.Must(template.New("report").Funcs(template.FuncMap{
		"formatRTT": formatRTTHTML,
		"rttClass":  rttClass,
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05 MST")
		},
	}).Parse(htmlTemplate))

	return &HTMLFormatter{
		config:   config,
		template: tmpl,
	}
}

// Format formats the trace result as an HTML report.
func (f *HTMLFormatter) Format(result *trace.TraceResult) ([]byte, error) {
	data := f.prepareData(result)

	var buf bytes.Buffer
	if err := f.template.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// htmlData holds the data for the HTML template.
type htmlData struct {
	Title       string
	Target      string
	ResolvedIP  string
	Timestamp   time.Time
	ProbeMethod string
	Completed   bool
	Hops        []htmlHop
	Summary     htmlSummary
	GeneratedAt time.Time
}

// htmlHop represents a hop for HTML rendering.
type htmlHop struct {
	Number      int
	IP          string
	Hostname    string
	ASN         string
	Org         string
	Country     string
	City        string
	AvgRTT      string
	MinRTT      string
	MaxRTT      string
	Jitter      string
	LossPercent string
	Responded   bool
	RTTClass    string
}

// htmlSummary holds summary data for HTML.
type htmlSummary struct {
	TotalHops   int
	Responding  int
	TotalTime   string
	PacketLoss  string
	Status      string
	StatusClass string
}

// prepareData converts TraceResult to template data.
func (f *HTMLFormatter) prepareData(result *trace.TraceResult) *htmlData {
	data := &htmlData{
		Title:       fmt.Sprintf("Traceroute to %s", result.Target),
		Target:      result.Target,
		ResolvedIP:  result.ResolvedIP.String(),
		Timestamp:   result.Timestamp,
		ProbeMethod: result.ProbeMethod,
		Completed:   result.Completed,
		Hops:        make([]htmlHop, len(result.Hops)),
		GeneratedAt: time.Now(),
	}

	responding := 0
	for i, hop := range result.Hops {
		h := htmlHop{
			Number:    hop.Number,
			Responded: hop.Responded,
		}

		if hop.Responded {
			responding++
			if hop.IP != nil {
				h.IP = hop.IP.String()
			}
			h.Hostname = hop.Hostname
			h.AvgRTT = formatRTTHTML(hop.AvgRTT)
			h.MinRTT = formatRTTHTML(hop.MinRTT)
			h.MaxRTT = formatRTTHTML(hop.MaxRTT)
			h.Jitter = formatRTTHTML(hop.Jitter)
			h.LossPercent = fmt.Sprintf("%.0f%%", hop.LossPercent)
			h.RTTClass = rttClass(hop.AvgRTT)

			if hop.ASN != nil {
				h.ASN = fmt.Sprintf("AS%d", hop.ASN.Number)
				h.Org = hop.ASN.Org
			}

			if hop.Geo != nil {
				h.Country = hop.Geo.CountryCode
				h.City = hop.Geo.City
			}
		} else {
			h.IP = "*"
			h.AvgRTT = "*"
			h.MinRTT = "*"
			h.MaxRTT = "*"
			h.LossPercent = "100%"
			h.RTTClass = "timeout"
		}

		data.Hops[i] = h
	}

	// Summary
	data.Summary = htmlSummary{
		TotalHops:  result.Summary.TotalHops,
		Responding: responding,
		TotalTime:  fmt.Sprintf("%.2f ms", result.Summary.TotalTimeMs),
		PacketLoss: fmt.Sprintf("%.1f%%", result.Summary.PacketLossPercent),
	}

	if result.Completed {
		data.Summary.Status = "Complete"
		data.Summary.StatusClass = "success"
	} else {
		data.Summary.Status = "Incomplete"
		data.Summary.StatusClass = "warning"
	}

	return data
}

// formatRTTHTML formats RTT for HTML display.
func formatRTTHTML(rtt float64) string {
	if rtt <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f", rtt)
}

// rttClass returns CSS class based on RTT value.
func rttClass(rtt float64) string {
	if rtt <= 0 {
		return "neutral"
	}
	switch {
	case rtt < 50:
		return "good"
	case rtt < 150:
		return "medium"
	default:
		return "bad"
	}
}

// ContentType returns the MIME type for HTML output.
func (f *HTMLFormatter) ContentType() string {
	return "text/html"
}

// FileExtension returns the file extension for HTML output.
func (f *HTMLFormatter) FileExtension() string {
	return "html"
}

// HTML template
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Poros Report</title>
    <style>
        :root {
            --bg-primary: #1a1b26;
            --bg-secondary: #24283b;
            --bg-tertiary: #414868;
            --text-primary: #c0caf5;
            --text-secondary: #a9b1d6;
            --text-muted: #565f89;
            --accent: #7aa2f7;
            --success: #9ece6a;
            --warning: #e0af68;
            --error: #f7768e;
            --border: #3b4261;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
            padding: 2rem;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        header {
            text-align: center;
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--border);
        }

        h1 {
            color: var(--accent);
            font-size: 2rem;
            margin-bottom: 0.5rem;
        }

        .subtitle {
            color: var(--text-muted);
            font-size: 0.9rem;
        }

        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }

        .info-card {
            background: var(--bg-secondary);
            padding: 1rem;
            border-radius: 8px;
            border: 1px solid var(--border);
        }

        .info-card label {
            color: var(--text-muted);
            font-size: 0.8rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .info-card value {
            display: block;
            color: var(--text-primary);
            font-size: 1.1rem;
            font-weight: 500;
            margin-top: 0.25rem;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            background: var(--bg-secondary);
            border-radius: 8px;
            overflow: hidden;
            margin-bottom: 2rem;
        }

        th, td {
            padding: 0.75rem 1rem;
            text-align: left;
            border-bottom: 1px solid var(--border);
        }

        th {
            background: var(--bg-tertiary);
            color: var(--text-secondary);
            font-weight: 600;
            font-size: 0.85rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        tr:last-child td {
            border-bottom: none;
        }

        tr:hover {
            background: var(--bg-tertiary);
        }

        .hop-num {
            color: var(--accent);
            font-weight: 600;
        }

        .ip {
            font-family: 'Monaco', 'Menlo', monospace;
            color: var(--text-primary);
        }

        .hostname {
            color: var(--success);
        }

        .asn {
            color: var(--warning);
            font-size: 0.85rem;
        }

        .geo {
            color: var(--text-muted);
            font-size: 0.85rem;
        }

        .rtt {
            font-family: 'Monaco', 'Menlo', monospace;
        }

        .rtt.good { color: var(--success); }
        .rtt.medium { color: var(--warning); }
        .rtt.bad { color: var(--error); }
        .rtt.timeout { color: var(--error); }
        .rtt.neutral { color: var(--text-muted); }

        .loss {
            font-size: 0.85rem;
        }

        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 1rem;
            background: var(--bg-secondary);
            padding: 1.5rem;
            border-radius: 8px;
            border: 1px solid var(--border);
        }

        .summary-item {
            text-align: center;
        }

        .summary-item .value {
            font-size: 1.5rem;
            font-weight: 600;
            color: var(--accent);
        }

        .summary-item .label {
            color: var(--text-muted);
            font-size: 0.8rem;
            text-transform: uppercase;
        }

        .status.success { color: var(--success); }
        .status.warning { color: var(--warning); }

        footer {
            text-align: center;
            margin-top: 2rem;
            padding-top: 1rem;
            border-top: 1px solid var(--border);
            color: var(--text-muted);
            font-size: 0.8rem;
        }

        @media (max-width: 768px) {
            body { padding: 1rem; }
            h1 { font-size: 1.5rem; }
            th, td { padding: 0.5rem; font-size: 0.85rem; }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🔍 {{.Title}}</h1>
            <p class="subtitle">Generated by Poros Network Path Tracer</p>
        </header>

        <div class="info-grid">
            <div class="info-card">
                <label>Target</label>
                <value>{{.Target}}</value>
            </div>
            <div class="info-card">
                <label>Resolved IP</label>
                <value>{{.ResolvedIP}}</value>
            </div>
            <div class="info-card">
                <label>Probe Method</label>
                <value>{{.ProbeMethod | html}}</value>
            </div>
            <div class="info-card">
                <label>Timestamp</label>
                <value>{{formatTime .Timestamp}}</value>
            </div>
        </div>

        <table>
            <thead>
                <tr>
                    <th>Hop</th>
                    <th>IP Address</th>
                    <th>Hostname</th>
                    <th>ASN</th>
                    <th>Location</th>
                    <th>Avg RTT</th>
                    <th>Min</th>
                    <th>Max</th>
                    <th>Loss</th>
                </tr>
            </thead>
            <tbody>
                {{range .Hops}}
                <tr>
                    <td class="hop-num">{{.Number}}</td>
                    <td class="ip">{{.IP}}</td>
                    <td class="hostname">{{if .Hostname}}{{.Hostname}}{{else}}-{{end}}</td>
                    <td class="asn">{{if .ASN}}{{.ASN}}<br><small>{{.Org}}</small>{{else}}-{{end}}</td>
                    <td class="geo">{{if .City}}{{.City}}, {{end}}{{if .Country}}{{.Country}}{{else}}-{{end}}</td>
                    <td class="rtt {{.RTTClass}}">{{.AvgRTT}}{{if .Responded}} ms{{end}}</td>
                    <td class="rtt neutral">{{.MinRTT}}</td>
                    <td class="rtt neutral">{{.MaxRTT}}</td>
                    <td class="loss">{{.LossPercent}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <div class="summary">
            <div class="summary-item">
                <div class="value">{{.Summary.TotalHops}}</div>
                <div class="label">Total Hops</div>
            </div>
            <div class="summary-item">
                <div class="value">{{.Summary.Responding}}</div>
                <div class="label">Responding</div>
            </div>
            <div class="summary-item">
                <div class="value">{{.Summary.TotalTime}}</div>
                <div class="label">Total Time</div>
            </div>
            <div class="summary-item">
                <div class="value">{{.Summary.PacketLoss}}</div>
                <div class="label">Packet Loss</div>
            </div>
            <div class="summary-item">
                <div class="value status {{.Summary.StatusClass}}">{{.Summary.Status}}</div>
                <div class="label">Status</div>
            </div>
        </div>

        <footer>
            <p>Generated by <strong>Poros</strong> on {{formatTime .GeneratedAt}}</p>
            <p>https://github.com/KilimcininKorOglu/poros</p>
        </footer>
    </div>
</body>
</html>
`
