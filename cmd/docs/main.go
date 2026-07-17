package main

import (
	"github.com/bakito/docs-gen/docs"
	"github.com/bakito/docs-gen/pkg/cli"
	"github.com/bakito/docs-gen/pkg/cobra"
	"github.com/bakito/docs-gen/pkg/template"
	"github.com/bakito/docs-gen/pkg/yaml"
	"github.com/bakito/kubexporter/pkg/export"
	"github.com/bakito/kubexporter/pkg/types"
)

const (
	cliStartMarker     = "<!-- cli-doc-start -->"
	cliEndMarker       = "<!-- cli-doc-end -->"
	yamlStartMarker    = "<!-- yaml-doc-start -->"
	yamlEndMarker      = "<!-- yaml-doc-end -->"
	cobraStartMarker   = "// cobra-doc-start"
	cobraEndMarker     = "// cobra-doc-end"
	metricsStartMarker = "<!-- metrics-doc-start -->"
	metricsEndMarker   = "<!-- metrics-doc-end -->"

	checkMetricsStartMarker = "  # metrics-doc-start"
	checkMetricsEndtMarker  = "  # metrics-doc-end"
)

func main() {
	docs.UpdateDocumentation("cmd/zz_generated_docs.go",
		cobra.UpdateDocumentation[types.Config](cobraStartMarker, cobraEndMarker),
	)
	docs.UpdateDocumentation("README.md",
		cli.UpdateDocumentation(cliStartMarker, cliEndMarker, ".", "go", "run", ".", "--help"),
		yaml.UpdateDocumentation[types.Config](yamlStartMarker, yamlEndMarker),
		template.UpdateDocumentation(export.MetricDefinitions(), metricsStartMarker, metricsEndMarker,
			"| {{ .Key }} | {{ .Value }} |\n",
			template.WithPrefix("| Metric | Description |\n| ------ | ----------- |\n"),
		),
	)
	docs.UpdateDocumentation("testdata/e2e/verify-metrics.sh",
		template.UpdateDocumentation(export.MetricDefinitions(), checkMetricsStartMarker, checkMetricsEndtMarker,
			"  {{ .Key }}\n",
		),
	)
}
