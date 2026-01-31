package slackmd

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// formatTable renders rows as a box-drawing ASCII table.
//
//	+-------+-------+
//	| col1  | col2  |
//	+-------+-------+
//	| val1  | val2  |
//	+-------+-------+
func formatTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	colWidths := tableColumnWidths(rows)

	var b strings.Builder
	rule := tableRule(colWidths)

	b.WriteString(rule)
	for i, row := range rows {
		b.WriteByte('|')
		for j, cell := range row {
			pad := colWidths[j] - utf8.RuneCountInString(cell)
			fmt.Fprintf(&b, " %s%s |", cell, strings.Repeat(" ", pad))
		}
		b.WriteByte('\n')
		if i == 0 || i == len(rows)-1 {
			b.WriteString(rule)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func tableRule(colWidths []int) string {
	var b strings.Builder
	b.WriteByte('+')
	for _, w := range colWidths {
		b.WriteString(strings.Repeat("-", w+2))
		b.WriteByte('+')
	}
	b.WriteByte('\n')
	return b.String()
}

func tableColumnWidths(rows [][]string) []int {
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	widths := make([]int, maxCols)
	for _, row := range rows {
		for j, cell := range row {
			w := utf8.RuneCountInString(cell)
			if w > widths[j] {
				widths[j] = w
			}
		}
	}
	return widths
}
