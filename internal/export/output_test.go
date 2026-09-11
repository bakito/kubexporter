package export

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-runewidth"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/bakito/kubexporter/internal/export/worker"
	"github.com/bakito/kubexporter/internal/types"
)

// captureStdout collects everything the given function prints to stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = orig
	}()

	f()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func testExporter(t *testing.T, progress types.Progress) *exporter {
	t.Helper()
	format := types.DefaultFormat
	config := types.NewConfig(nil, &genericclioptions.PrintFlags{
		OutputFormat:       &format,
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	})
	config.Target = "exports"
	config.Progress = progress

	return &exporter{
		config: config,
		l:      config.Logger(),
		stats:  &worker.Stats{Kinds: 42, Pages: 7, Resources: 1234},
		start:  time.Now(),
	}
}

func valueColumns(t *testing.T, out string) []int {
	t.Helper()
	var columns []int
	for line := range strings.SplitSeq(strings.Trim(out, "\n"), "\n") {
		// the value is the last token of an entry line, headings are ignored
		idx := strings.LastIndex(strings.TrimRight(line, " "), "  ") + 2
		if idx <= 1 {
			continue
		}
		columns = append(columns, runewidth.StringWidth(line[:idx]))
	}
	return columns
}

func TestPrintEntriesAlignsValues(t *testing.T) {
	entries := []entry{
		{"📜", "kinds", "42"},
		{"📚", "exported resources", "1234"},
		{"🏠", "namespaces", "3"},
		{"🔖", "context", ""},
	}
	// the icons have no reliable display width, so the alignment is verified without them
	config := types.NewConfig(nil, nil)
	config.Progress = types.ProgressSimple
	l := config.Logger()

	out := captureStdout(t, func() {
		printEntries(l, entries)
	})

	if strings.Contains(out, "✓") {
		t.Errorf("expected no check icon per line, but got:\n%s", out)
	}

	if strings.Contains(out, "context") {
		t.Errorf("expected entries without a value to be skipped, but got:\n%s", out)
	}

	columns := valueColumns(t, out)
	if len(columns) != len(entries)-1 {
		t.Fatalf("expected %d lines, but got %d: %q", len(entries)-1, len(columns), out)
	}
	for i, c := range columns {
		if c != columns[0] {
			t.Errorf("value of line %d starts at %d, but expected %d:\n%s", i, c, columns[0], out)
		}
	}
}

// TestEntryIconsHaveEmojiPresentation makes sure all icons occupy two cells in the terminal.
// Emoji requiring a variation selector (U+FE0F) are rendered one cell wide by many
// terminals, which would break the column alignment.
func TestEntryIconsHaveEmojiPresentation(t *testing.T) {
	e := testExporter(t, types.ProgressBar)
	e.config.Archive = true
	e.config.ArchiveRetentionDays = 7
	e.config.PrintSize = true
	e.config.CreatedWithin = time.Minute
	e.archive = "exports.tar.gz"
	e.stats.Errors = 1

	entries := append(e.introEntries(), e.statsEntries()...)
	for _, en := range entries {
		if strings.ContainsRune(en.icon, '\ufe0f') {
			t.Errorf("icon %q of %q must not use a variation selector", en.icon, en.label)
		}
	}
}

func TestWriteIntro(t *testing.T) {
	e := testExporter(t, types.ProgressBar)
	e.config.Summary = true
	e.config.PrintSize = true
	e.config.Worker = 4

	out := captureStdout(t, e.writeIntro)

	wants := []string{"kubexporter", "Configuration", "target", "exports", "worker", "Exporting ..."}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("expected the intro to contain %q, but got:\n%s", want, out)
		}
	}
}

func TestPrintStatsSimpleHasNoIcons(t *testing.T) {
	e := testExporter(t, types.ProgressSimple)

	out := captureStdout(t, e.printStats)

	if !strings.Contains(out, "exported resources") {
		t.Errorf("expected the stats to contain the exported resources, but got:\n%s", out)
	}
	for _, icon := range []string{"📜", "📚", "🏠", "⏳"} {
		if strings.Contains(out, icon) {
			t.Errorf("expected no icon %q in simple mode, but got:\n%s", icon, out)
		}
	}
	// the values must still be aligned
	columns := valueColumns(t, out)
	if len(columns) < 2 {
		t.Fatalf("expected multiple entry lines, but got:\n%s", out)
	}
	for i, c := range columns {
		if c != columns[0] {
			t.Errorf("value of line %d starts at %d, but expected %d:\n%s", i, c, columns[0], out)
		}
	}
}
