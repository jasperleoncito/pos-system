package export

import (
	"encoding/csv"
	"io"
)

// CSVExporter renders with the stdlib csv writer.
type CSVExporter struct{}

func (CSVExporter) ContentType() string { return "text/csv; charset=utf-8" }
func (CSVExporter) FileExt() string     { return "csv" }

func (CSVExporter) Write(w io.Writer, doc *Document) error {
	cw := csv.NewWriter(w)

	header := make([]string, len(doc.Columns))
	for i, col := range doc.Columns {
		header[i] = col.Label
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, row := range doc.Rows {
		record := make([]string, len(doc.Columns))
		for i, col := range doc.Columns {
			record[i] = cellString(row[col.Key], col.Kind)
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}

	if doc.Totals != nil {
		record := make([]string, len(doc.Columns))
		for i, col := range doc.Columns {
			if v, ok := doc.Totals[col.Key]; ok {
				record[i] = cellString(v, col.Kind)
			}
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}

	cw.Flush()
	return cw.Error()
}
