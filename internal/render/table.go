package render

import (
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// Table creates a new table with the default kubexporter rendition, writing to stdout.
// Additional options can be provided to customize the table.
func Table(opts ...tablewriter.Option) *tablewriter.Table {
	return TableTo(os.Stdout, opts...)
}

// TableTo creates a new table with the default kubexporter rendition, writing to the given writer.
// Additional options can be provided to customize the table.
func TableTo(w io.Writer, opts ...tablewriter.Option) *tablewriter.Table {
	rendition := tw.Rendition{
		Settings: tw.Settings{
			Lines:      tw.Lines{ShowHeaderLine: tw.Off, ShowFooterLine: tw.Off},
			Separators: tw.Separators{BetweenRows: tw.Off, BetweenColumns: tw.Off},
		},
	}
	options := []tablewriter.Option{
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRenderer(renderer.NewBlueprint(rendition)),
	}

	return tablewriter.NewTable(w, append(options, opts...)...)
}
