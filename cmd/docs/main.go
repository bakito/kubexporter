package main

import (
	"github.com/bakito/docs-gen/docs"
	"github.com/bakito/docs-gen/pkg/cli"
	"github.com/bakito/docs-gen/pkg/yaml"
	"github.com/bakito/kubexporter/pkg/types"
)

const (
	cliStartMarker  = "<!-- cli-doc-start -->"
	cliEndMarker    = "<!-- cli-doc-end -->"
	yamlStartMarker = "<!-- yaml-doc-start -->"
	yamlEndMarker   = "<!-- yaml-doc-end -->"
)

func main() {
	docs.UpdateDocumentation("README.md",
		cli.UpdateDocumentation(cliStartMarker, cliEndMarker, ".", "go", "run", ".", "--help"),
		yaml.UpdateDocumentation[types.Config](yamlStartMarker, yamlEndMarker),
	)
}
