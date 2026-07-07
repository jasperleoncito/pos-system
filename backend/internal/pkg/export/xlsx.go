package export

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// XLSXExporter renders with excelize; money cells become numeric pesos.
type XLSXExporter struct{}

func (XLSXExporter) ContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}
func (XLSXExporter) FileExt() string { return "xlsx" }

func (XLSXExporter) Write(w io.Writer, doc *Document) error {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)

	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	money, err := f.NewStyle(&excelize.Style{NumFmt: 4}) // #,##0.00
	if err != nil {
		return err
	}

	// Title + subtitle + header.
	_ = f.SetCellValue(sheet, "A1", doc.Title)
	_ = f.SetCellStyle(sheet, "A1", "A1", bold)
	_ = f.SetCellValue(sheet, "A2", doc.Subtitle)

	headerRow := 4
	for i, col := range doc.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		_ = f.SetCellValue(sheet, cell, col.Label)
		_ = f.SetCellStyle(sheet, cell, cell, bold)
		width := 14.0
		if len(col.Label) > 12 {
			width = float64(len(col.Label)) + 4
		}
		name, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, name, name, width)
	}

	writeRow := func(rowIdx int, values map[string]any, styled bool) {
		for i, col := range doc.Columns {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowIdx)
			v, ok := values[col.Key]
			if !ok || v == nil {
				continue
			}
			if col.Kind == KindMoney {
				if c, isNum := toInt64(v); isNum {
					_ = f.SetCellValue(sheet, cell, float64(c)/100)
					_ = f.SetCellStyle(sheet, cell, cell, money)
					continue
				}
			}
			_ = f.SetCellValue(sheet, cell, fmt.Sprintf("%v", v))
			if styled {
				_ = f.SetCellStyle(sheet, cell, cell, bold)
			}
		}
	}

	for i, row := range doc.Rows {
		writeRow(headerRow+1+i, row, false)
	}
	if doc.Totals != nil {
		writeRow(headerRow+1+len(doc.Rows), doc.Totals, true)
	}

	return f.Write(w)
}
