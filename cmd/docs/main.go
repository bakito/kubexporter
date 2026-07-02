// Update the README.md file with the generated CLI documentation
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

const (
	cliStartMarker = "<!-- cli-doc-start -->"
	cliEndMarker   = "<!-- cli-doc-end -->"
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
