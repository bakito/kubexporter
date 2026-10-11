package render

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// Tbl represents a kubectl-style table builder using text/tabwriter.
type Tbl struct {
	w      io.Writer
	header []string
	rows   [][]string
	footer []string
}

// Table creates a new kubectl-style table writing to stdout.
// Use Header(), Append(), Footer(), and Render() to build and output the table.
func Table() *Tbl {
	return TableTo(os.Stdout)
}

// TableTo creates a new kubectl-style table writing to the given writer.
func TableTo(w io.Writer) *Tbl {
	return &Tbl{w: w}
}

// Header sets the table header columns. All strings are converted to uppercase.
func (t *Tbl) Header(cols ...string) {
	upper := make([]string, len(cols))
	for i, c := range cols {
		upper[i] = strings.ToUpper(c)
	}
	t.header = upper
}

// Append adds a row to the table.
func (t *Tbl) Append(cols ...string) {
	t.rows = append(t.rows, cols)
}

// Footer sets the table footer row.
func (t *Tbl) Footer(cols ...string) {
	t.footer = cols
}

// Render writes the table to the configured writer in kubectl style.
func (t *Tbl) Render() error {
	w := tabwriter.NewWriter(t.w, 0, 0, 2, ' ', 0)

	// Write header
	if t.header != nil {
		fmt.Fprintln(w, strings.Join(t.header, "\t"))
	}

	// Write rows
	for _, row := range t.rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	// Write footer
	if t.footer != nil {
		fmt.Fprintln(w, strings.Join(t.footer, "\t"))
	}

	return w.Flush()
}
