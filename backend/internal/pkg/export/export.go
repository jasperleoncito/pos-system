// Package export renders tabular report documents as CSV, XLSX, or PDF
// behind one Exporter interface. Money cells are integer centavos.
package export

import (
	"fmt"
	"io"
)

// Column kinds control per-format cell rendering.
const (
	KindText  = "text"
	KindMoney = "money" // centavos → pesos
	KindNum   = "number"
)

type Column struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

// Document is one renderable report.
type Document struct {
	Title    string           `json:"title"`
	Subtitle string           `json:"subtitle"` // business + range
	Columns  []Column         `json:"columns"`
	Rows     []map[string]any `json:"rows"`
	Totals   map[string]any   `json:"totals,omitempty"` // keyed like rows
	LogoPNG  []byte           `json:"-"`                // optional PDF header logo
}

// Exporter renders a document into one file format.
type Exporter interface {
	ContentType() string
	FileExt() string
	Write(w io.Writer, doc *Document) error
}

// ForFormat picks the exporter; ok is false for unknown formats.
func ForFormat(format string) (Exporter, bool) {
	switch format {
	case "csv":
		return CSVExporter{}, true
	case "xlsx":
		return XLSXExporter{}, true
	case "pdf":
		return PDFExporter{}, true
	}
	return nil, false
}

// cellString renders a value for text-based formats.
func cellString(v any, kind string) string {
	if v == nil {
		return ""
	}
	if kind == KindMoney {
		if c, ok := toInt64(v); ok {
			return fmt.Sprintf("%.2f", float64(c)/100)
		}
	}
	return fmt.Sprintf("%v", v)
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	}
	return 0, false
}
