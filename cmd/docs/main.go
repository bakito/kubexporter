// Update the README.md file with the generated CLI documentation
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/bakito/kubexporter/pkg/types"
)

const (
	cliStartMarker  = "<!-- cli-doc-start -->"
	cliEndMarker    = "<!-- cli-doc-end -->"
	yamlStartMarker = "<!-- yaml-doc-start -->"
	yamlEndMarker   = "<!-- yaml-doc-end -->"
	tagDocs         = "docs"
)

func main() {
	slog.Info("Reading README.md")
	content, err := os.ReadFile("README.md")
	if err != nil {
		slog.Error("Error reading README.md", "error", err)
		os.Exit(1)
	}

	fileContent := string(content)

	slog.Info("Generating cli docs")
	fileContent = generateCLiDocumentation(fileContent)

	slog.Info("Generating yaml configuration")
	fileContent = generateYAMLDocumentation(fileContent)

	slog.Info("Writing README.md")
	err = os.WriteFile("README.md", []byte(fileContent), 0o644)
	if err != nil {
		slog.Error("Error writing README.md", "error", err)
		os.Exit(1)
	}
}

func generateCLiDocumentation(fileContent string) string {
	var buf strings.Builder
	buf.WriteString("```\n")
	writeCliDocumentation(&buf)
	buf.WriteString("```\n")
	return updateDocumentationSection(fileContent, cliStartMarker, cliEndMarker, buf.String())
}

func updateDocumentationSection(fileContent, startMarker, endMarker, newContent string) string {
	startIdx := strings.Index(fileContent, startMarker)
	endIdx := strings.Index(fileContent, endMarker)

	if startIdx == -1 || endIdx == -1 {
		slog.Error(fmt.Sprintf("Could not find markers %s and %s in README.md", startMarker, endMarker))
		os.Exit(1)
	}

	return fileContent[:startIdx+len(startMarker)] + "\n" + newContent + fileContent[endIdx:]
}

func writeCliDocumentation(w io.Writer) {
	cmd := exec.CommandContext(context.Background(), "go", "run", ".", "--help")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		slog.Error("Error executing main.go with --help", "error", err)
		os.Exit(1)
	}
	if _, err := w.Write(output); err != nil {
		slog.Error("Error writing CLI documentation", "error", err)
		os.Exit(1)
	}
}

func generateYAMLDocumentation(fileContent string) string {
	var buf strings.Builder
	buf.WriteString("```yaml\n")
	writeYAMLDocumentation(&buf, reflect.TypeFor[types.Config](), "", "")
	buf.WriteString("```\n")

	return updateDocumentationSection(fileContent, yamlStartMarker, yamlEndMarker, buf.String())
}

func writeYAMLDocumentation(w io.Writer, t reflect.Type, firstPrefix, otherPrefix string) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	var i int
	for _, field := range reflect.VisibleFields(t) {
		if field.PkgPath != "" {
			continue
		}

		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "-" {
			continue
		}
		yamlTag = strings.TrimSuffix(yamlTag, ",omitempty")

		ft := field.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		pf := otherPrefix
		if i == 0 {
			pf = firstPrefix
		}

		newFirstPrefix := pf + "  "
		newOtherPrefix := otherPrefix + "  "

		if yamlTag == "replicas" && ft.Kind() == reflect.Slice {
			ft = ft.Elem()
			newFirstPrefix += "- "
			newOtherPrefix += "  "
		}

		if yamlTag != "" {
			docs := field.Tag.Get(tagDocs)
			fieldType := fieldTypeString(ft)
			fmt.Fprintf(w, "%s# %s (%s)\n", pf, docs, fieldType)
			fmt.Fprintf(w, "%s%s:\n", pf, yamlTag)
			i++
		}

		if ft.Kind() == reflect.Struct && ft.Name() != "Time" {
			writeYAMLDocumentation(w, ft, newFirstPrefix, newOtherPrefix)
		}
	}
}

func fieldTypeString(ft reflect.Type) string {
	if ft.Kind() == reflect.Map {
		return fmt.Sprintf("map[%s:%s]", ft.Key().Kind().String(), fieldTypeString(ft.Elem()))
	} else if ft.Kind() == reflect.Slice {
		return "[]" + fieldTypeString(ft.Elem())
	}
	return ft.Kind().String()
}
