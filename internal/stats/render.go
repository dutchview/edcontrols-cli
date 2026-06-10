package stats

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// RenderTable writes the aggregation result as an aligned table with a
// trailing TOTAL row.
func RenderTable(out io.Writer, res Result) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	headers := make([]string, 0, len(res.GroupBy)+2)
	for _, name := range res.GroupBy {
		headers = append(headers, strings.ToUpper(name))
	}
	headers = append(headers, "COUNT", "OVERDUE")
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, g := range res.Groups {
		row := make([]string, 0, len(res.GroupBy)+2)
		for _, name := range res.GroupBy {
			row = append(row, g.Keys[name])
		}
		row = append(row, fmt.Sprintf("%d", g.Count), fmt.Sprintf("%d", g.Overdue))
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	total := make([]string, 0, len(res.GroupBy)+2)
	total = append(total, "TOTAL")
	for i := 1; i < len(res.GroupBy); i++ {
		total = append(total, "")
	}
	total = append(total, fmt.Sprintf("%d", res.Total), fmt.Sprintf("%d", res.Overdue))
	fmt.Fprintln(w, strings.Join(total, "\t"))

	w.Flush()
}
