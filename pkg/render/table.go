package render

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

func Table() *tablewriter.Table {
	rendition := tw.Rendition{
		Settings: tw.Settings{
			Lines:      tw.Lines{ShowHeaderLine: tw.Off, ShowFooterLine: tw.Off},
			Separators: tw.Separators{BetweenRows: tw.Off, BetweenColumns: tw.Off},
		},
	}
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRenderer(renderer.NewBlueprint(rendition)),
	)

	return table
}
