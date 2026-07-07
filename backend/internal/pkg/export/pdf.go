package export

import (
	"fmt"
	"io"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// PDFExporter renders with maroto v2; the tenant logo heads the page.
type PDFExporter struct{}

func (PDFExporter) ContentType() string { return "application/pdf" }
func (PDFExporter) FileExt() string     { return "pdf" }

func (PDFExporter) Write(w io.Writer, doc *Document) error {
	cfg := config.NewBuilder().
		WithLeftMargin(10).WithRightMargin(10).WithTopMargin(10).
		Build()
	m := maroto.New(cfg)

	// ---- header: logo + title ----
	headerCols := []core.Col{}
	titleGrid := 12
	if len(doc.LogoPNG) > 0 {
		headerCols = append(headerCols,
			image.NewFromBytesCol(2, doc.LogoPNG, extension.Png, props.Rect{Center: true, Percent: 90}))
		titleGrid = 10
	}
	headerCols = append(headerCols, col.New(titleGrid).Add(
		text.New(doc.Title, props.Text{Size: 14, Style: fontstyle.Bold, Top: 1}),
		text.New(doc.Subtitle, props.Text{Size: 9, Top: 8, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
	))
	m.AddRows(row.New(16).Add(headerCols...))
	m.AddRows(row.New(4).Add(col.New(12).Add(line.New())))

	// ---- table ----
	grids := columnGrids(len(doc.Columns))
	headerCells := make([]core.Col, len(doc.Columns))
	for i, c := range doc.Columns {
		headerCells[i] = text.NewCol(grids[i], c.Label, props.Text{
			Size: 8, Style: fontstyle.Bold, Align: alignFor(c.Kind),
		})
	}
	m.AddRows(row.New(7).Add(headerCells...))

	for _, r := range doc.Rows {
		cells := make([]core.Col, len(doc.Columns))
		for i, c := range doc.Columns {
			cells[i] = text.NewCol(grids[i], pdfCell(r[c.Key], c.Kind), props.Text{
				Size: 8, Align: alignFor(c.Kind),
			})
		}
		m.AddRows(row.New(6).Add(cells...))
	}

	if doc.Totals != nil {
		m.AddRows(row.New(2).Add(col.New(12).Add(line.New())))
		cells := make([]core.Col, len(doc.Columns))
		for i, c := range doc.Columns {
			value := ""
			if v, ok := doc.Totals[c.Key]; ok {
				value = pdfCell(v, c.Kind)
			}
			cells[i] = text.NewCol(grids[i], value, props.Text{
				Size: 8, Style: fontstyle.Bold, Align: alignFor(c.Kind),
			})
		}
		m.AddRows(row.New(7).Add(cells...))
	}

	rendered, err := m.Generate()
	if err != nil {
		return fmt.Errorf("failed to render pdf: %w", err)
	}
	_, err = w.Write(rendered.GetBytes())
	return err
}

// columnGrids spreads maroto's 12-column grid across the table, giving
// leftover width to the first (usually descriptive) column.
func columnGrids(n int) []int {
	if n <= 0 {
		return nil
	}
	if n > 12 {
		n = 12
	}
	base := 12 / n
	rest := 12 % n
	grids := make([]int, n)
	for i := range grids {
		grids[i] = base
	}
	grids[0] += rest
	return grids
}

func alignFor(kind string) align.Type {
	if kind == KindMoney || kind == KindNum {
		return align.Right
	}
	return align.Left
}

func pdfCell(v any, kind string) string {
	if kind == KindMoney {
		if c, ok := toInt64(v); ok {
			return fmt.Sprintf("P%.2f", float64(c)/100) // gofpdf core fonts lack ₱
		}
	}
	return cellString(v, kind)
}
