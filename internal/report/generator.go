package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

type Report struct {
	Title       string
	Type        string
	GeneratedAt time.Time
	Headers     []string
	Rows        [][]string
	Summary     map[string]string
}

// Generate writes the requested formats to outputDir and returns their paths.
// Pass formats as "csv", "html", "pdf" — or leave empty to generate all three.
func Generate(r *Report, outputDir string, formats ...string) (map[string]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	if len(formats) == 0 {
		formats = []string{"csv", "html", "pdf"}
	}

	want := map[string]bool{}
	for _, f := range formats {
		want[strings.ToLower(strings.TrimSpace(f))] = true
	}

	base := fmt.Sprintf("%s_%d", sanitize(r.Type), r.GeneratedAt.Unix())
	paths := map[string]string{}

	if want["csv"] {
		p := filepath.Join(outputDir, base+".csv")
		if err := writeCSV(r, p); err != nil {
			return nil, fmt.Errorf("CSV: %w", err)
		}
		paths["csv"] = p
	}

	if want["html"] {
		p := filepath.Join(outputDir, base+".html")
		if err := writeHTML(r, p); err != nil {
			return nil, fmt.Errorf("HTML: %w", err)
		}
		paths["html"] = p
	}

	if want["pdf"] {
		p := filepath.Join(outputDir, base+".pdf")
		if err := writePDF(r, p); err != nil {
			return nil, fmt.Errorf("PDF: %w", err)
		}
		paths["pdf"] = p
	}

	return paths, nil
}

func writeCSV(r *Report, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{r.Title})
	w.Write([]string{"Generated", r.GeneratedAt.Format(time.RFC1123)})
	w.Write([]string{})

	if len(r.Summary) > 0 {
		w.Write([]string{"Summary"})
		for k, v := range r.Summary {
			w.Write([]string{k, v})
		}
		w.Write([]string{})
	}

	w.Write(r.Headers)
	for _, row := range r.Rows {
		w.Write(row)
	}

	w.Flush()
	return w.Error()
}

const htmlTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{.Title}}</title>
<style>
  body { font-family: Arial, sans-serif; margin: 40px; color: #222; }
  h1   { color: #1a1a2e; border-bottom: 2px solid #f0a500; padding-bottom: 8px; }
  .meta { color: #666; font-size: 0.85em; margin-bottom: 24px; }
  .summary { background: #f9f9f9; border: 1px solid #ddd; border-radius: 6px;
             padding: 16px; margin-bottom: 24px; display: grid;
             grid-template-columns: max-content 1fr; gap: 6px 24px; }
  .summary dt { font-weight: bold; color: #444; }
  .summary dd { margin: 0; }
  table  { border-collapse: collapse; width: 100%; }
  th     { background: #1a1a2e; color: #f0a500; padding: 10px 12px; text-align: left; }
  td     { padding: 8px 12px; border-bottom: 1px solid #eee; }
  tr:nth-child(even) td { background: #f9f9f9; }
  tr:hover td { background: #fff3cd; }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<p class="meta">Report type: <strong>{{.Type}}</strong> &nbsp;|&nbsp; Generated: {{.GeneratedAt.Format "Mon, 02 Jan 2006 15:04:05 MST"}}</p>

{{if .Summary}}
<dl class="summary">
{{range $k,$v := .Summary}}<dt>{{$k}}</dt><dd>{{$v}}</dd>{{end}}
</dl>
{{end}}

<table>
  <thead><tr>{{range .Headers}}<th>{{.}}</th>{{end}}</tr></thead>
  <tbody>
    {{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>{{end}}
  </tbody>
</table>
</body>
</html>`

func writeHTML(r *Report, path string) error {
	t, err := template.New("report").Parse(htmlTmpl)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, r)
}

func writePDF(r *Report, path string) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 20, 15)
	pdf.AddPage()

	pageW, _ := pdf.GetPageSize()
	contentW := pageW - 30

	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(26, 26, 46)
	pdf.CellFormat(contentW, 10, r.Title, "", 1, "L", false, 0, "")

	pdf.SetDrawColor(240, 165, 0)
	pdf.SetLineWidth(0.8)
	pdf.Line(15, pdf.GetY()+1, pageW-15, pdf.GetY()+1)
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(contentW, 6,
		fmt.Sprintf("Report type: %s   |   Generated: %s", r.Type, r.GeneratedAt.Format("Mon, 02 Jan 2006 15:04:05")),
		"", 1, "L", false, 0, "")
	pdf.Ln(4)

	if len(r.Summary) > 0 {
		pdf.SetFillColor(249, 249, 249)
		pdf.SetDrawColor(200, 200, 200)
		pdf.SetLineWidth(0.3)
		pdf.SetFont("Arial", "B", 10)
		pdf.SetTextColor(50, 50, 50)
		pdf.CellFormat(contentW, 7, "Summary", "1", 1, "L", true, 0, "")
		pdf.SetFont("Arial", "", 9)
		for k, v := range r.Summary {
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(contentW/2, 6, k, "LR", 0, "L", true, 0, "")
			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(contentW/2, 6, v, "R", 1, "L", true, 0, "")
		}
		pdf.Ln(4)
	}

	if len(r.Headers) > 0 {
		colW := contentW / float64(len(r.Headers))
		pdf.SetFillColor(26, 26, 46)
		pdf.SetTextColor(240, 165, 0)
		pdf.SetFont("Arial", "B", 9)
		for _, h := range r.Headers {
			pdf.CellFormat(colW, 8, h, "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 9)
		fill := false
		for _, row := range r.Rows {
			if fill {
				pdf.SetFillColor(249, 249, 249)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			pdf.SetTextColor(50, 50, 50)
			for _, cell := range row {
				pdf.CellFormat(colW, 7, cell, "1", 0, "L", fill, 0, "")
			}
			pdf.Ln(-1)
			fill = !fill
		}
	}

	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 10, fmt.Sprintf("BoltQ Report Engine  |  Page %d", pdf.PageNo()), "", 0, "C", false, 0, "")

	return pdf.OutputFileAndClose(path)
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	var b bytes.Buffer
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		}
	}
	return b.String()
}
