package export

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/bakito/kubexporter/internal/client"
	"github.com/bakito/kubexporter/internal/export/metrics"
	"github.com/bakito/kubexporter/internal/export/progress"
	"github.com/bakito/kubexporter/internal/export/progress/bubbles"
	"github.com/bakito/kubexporter/internal/export/progress/mpb"
	"github.com/bakito/kubexporter/internal/export/progress/nop"
	"github.com/bakito/kubexporter/internal/export/worker"
	"github.com/bakito/kubexporter/internal/log"
	"github.com/bakito/kubexporter/internal/render"
	"github.com/bakito/kubexporter/internal/types"
	"github.com/bakito/kubexporter/version"
)

// enabled is the value used for entries that only represent an enabled flag.
const enabled = "enabled"

// indent is the indentation of all content printed below a heading.
const indent = "  "

// NewExporter create a new exporter.
func NewExporter(config *types.Config) (Exporter, error) {
	ac, err := client.NewAPIClient(config)
	if err != nil {
		return nil, err
	}

	return &exporter{
		config: config,
		ac:     ac,
		l:      config.Logger(),
		stats:  &worker.Stats{},
	}, nil
}

// Exporter interface.
type Exporter interface {
	Export(ctx context.Context) error
}

type exporter struct {
	start           time.Time
	l               log.YALI
	config          *types.Config
	stats           *worker.Stats
	archive         string
	deletedArchives []string
	ac              *client.APIClient
}

func (e *exporter) Export(ctx context.Context) error {
	e.start = time.Now()

	defer e.printStats()
	if e.config.ClearTarget {
		if err := e.purgeTarget(); err != nil {
			return err
		}
	}

	e.writeIntro()

	resources, err := e.listResources()
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		e.l.Printf("No resources found")
		return nil
	}

	slices.SortStableFunc(resources, func(a, b *types.GroupResource) int {
		ret := strings.Compare(a.APIGroup, b.APIGroup)
		if ret != 0 {
			return ret
		}
		return strings.Compare(a.APIResource.Kind, b.APIResource.Kind)
	})

	var prog progress.Progress

	switch e.config.Progress {
	case types.ProgressBar:
		prog = mpb.NewProgress(resources)
	case types.ProgressBarBubbles:
		prog = bubbles.NewProgress(resources)
	default:
		prog = nop.NewProgress()
	}

	var workers []worker.Worker
	for i := range e.config.Worker {
		workers = append(workers, worker.New(i, e.config, e.ac, prog))
	}

	var exportErr error
	var s *worker.Stats
	var done chan struct{}
	if prog.Async() {
		done = make(chan struct{})
		go func() {
			defer close(done)
			defer prog.Finish()
			s, exportErr = worker.RunExport(ctx, workers, resources)
			e.stats.Add(s)
		}()
		defer func() {
			<-done
		}()
	} else {
		s, exportErr = worker.RunExport(ctx, workers, resources)
		e.stats.Add(s)
		prog.Finish()
	}

	if err := prog.Run(); err != nil {
		return err
	}
	if prog.Async() {
		<-done
	}
	if exportErr != nil {
		return exportErr
	}

	if e.config.Summary {
		if err := e.printSummary(resources); err != nil {
			return err
		}
	}

	if e.config.Metrics != nil && e.config.Metrics.OTLP.Enabled {
		if err := metrics.SendOTLP(ctx, e, e.config.Metrics.OTLP, resources); err != nil {
			return err
		}
	}

	if e.config.Archive {
		err = e.tarGz()
		if err != nil {
			return err
		}

		if e.config.ArchiveRetentionDays > 0 {
			err = e.pruneArchives()
			if err != nil {
				return err
			}
		}

		if e.config.S3Config != nil {
			err = e.uploadS3(ctx)
			if err != nil {
				return err
			}
		}
		if e.config.GCSConfig != nil {
			err = e.uploadGCS(ctx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// entry is a single icon / label / value line of the intro or the final stats.
type entry struct {
	icon  string
	label string
	value string
}

// printEntries prints the entries as aligned label / value columns.
// Entries without a value are skipped. All icons are emoji with an own emoji
// presentation, occupying two cells; in simple mode they are omitted, so the
// columns stay aligned in both cases.
func printEntries(l log.YALI, entries []entry) {
	width := 0
	for _, en := range entries {
		if en.value == "" {
			continue
		}
		width = max(width, runewidth.StringWidth(en.label))
	}
	for _, en := range entries {
		if en.value == "" {
			continue
		}
		icon := ""
		if !l.Simple() {
			icon = en.icon + " "
		}
		pad := strings.Repeat(" ", max(width-runewidth.StringWidth(en.label), 0))
		l.Printf(indent+icon+en.label+pad+"  %s\n", en.value)
	}
}

func (e *exporter) writeIntro() {
	e.printHeading("🚀", "kubexporter "+version.Version)
	e.printHeading("🔧", "Configuration")
	printEntries(e.l, e.introEntries())
	e.l.Printf("\nExporting ...\n")
}

// printHeading prints a section heading underlined with a horizontal rule.
func (e *exporter) printHeading(icon, heading string) {
	rule := "─"
	width := runewidth.StringWidth(heading)
	if e.l.Simple() {
		rule = "-"
	} else if icon != "" {
		heading = icon + " " + heading
		// the icon is an emoji occupying two cells plus the separating space
		width += 3
	}
	e.l.Printf("\n%s\n%s\n", heading, strings.Repeat(rule, width))
}

func (e *exporter) introEntries() []entry {
	entries := []entry{
		{"🌐", "cluster", e.ClusterHost()},
	}
	if e.config.ContextName() != nil {
		entries = append(entries, entry{"🔖", "context", *e.config.ContextName()})
	}
	if !e.config.HasNamespaces() {
		entries = append(entries, entry{"🏠", "namespaces", "all"})
	} else {
		entries = append(entries, entry{"🏠", "namespaces", strings.Join(e.config.Namespaces, ", ")})
		if e.config.IncludeClusterResources {
			entries = append(entries, entry{"🌍", "cluster resources", "included"})
		}
	}
	entries = append(entries,
		entry{"📁", "target", e.config.Target},
		entry{"📜", "format", e.config.OutputFormat()},
	)
	if e.config.Worker > 1 {
		entries = append(entries, entry{"👷", "worker", strconv.Itoa(e.config.Worker)})
	}
	if e.config.Summary {
		entries = append(entries, entry{"📊", "summary", enabled})
	}
	if e.config.ConsiderOwnerReferences {
		entries = append(entries, entry{"👑", "owner references", "considered"})
	}
	if len(e.config.Masked.KindFields) > 0 {
		entries = append(entries, entry{"🤿", "masked fields", e.config.Masked.KindFields.String()})
	}
	if len(e.config.Encrypted.KindFields) > 0 {
		entries = append(entries, entry{"🔒", "encrypted fields", e.config.Encrypted.KindFields.String()})
	}
	if e.config.CreatedWithin > 0 {
		entries = append(entries, entry{"⏳", "created within", e.config.CreatedWithin.String()})
	}
	if e.config.AsLists {
		entries = append(entries, entry{"📦", "as lists", enabled})
	} else if e.config.QueryPageSize != 0 {
		entries = append(entries, entry{"📃", "query page size", strconv.Itoa(e.config.QueryPageSize)})
	}
	if e.config.PrintSize {
		entries = append(entries, entry{"📏", "print size", enabled})
	}
	entries = append(entries, e.introArchiveEntries()...)
	if e.config.Metrics != nil && e.config.Metrics.OTLP.Enabled {
		entries = append(entries, entry{"📈", "OTLP metrics", e.config.Metrics.OTLP.Endpoint})
	}
	return entries
}

func (e *exporter) introArchiveEntries() []entry {
	if !e.config.Archive {
		return nil
	}
	entries := []entry{{"💾", "archive", enabled}}
	if e.config.ArchiveRetentionDays > 0 {
		entries = append(entries, entry{
			"🚮", "archive retention",
			fmt.Sprintf("%d days", e.config.ArchiveRetentionDays),
		})
	}
	if e.config.S3Config != nil {
		entries = append(entries, entry{
			"🪣", "S3 upload",
			fmt.Sprintf("%s/%s", e.config.S3Config.Endpoint, e.config.S3Config.Bucket),
		})
	}
	if e.config.GCSConfig != nil {
		entries = append(entries, entry{"🪣", "GCS upload", e.config.GCSConfig.Bucket})
	}
	return entries
}

func (e *exporter) listResources() ([]*types.GroupResource, error) {
	lists, err := e.ac.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	var resources []*types.GroupResource

	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			r := &types.GroupResource{
				APIGroup:        gv.Group,
				APIVersion:      gv.Version,
				APIGroupVersion: gv.String(),
				APIResource:     resource,
			}
			if !allowsList(resource) ||
				e.config.IsExcluded(r) ||
				(!resource.Namespaced && e.config.HasNamespaces() && !e.config.IncludeClusterResources) {
				continue
			}

			resources = append(resources, r)
		}
	}
	return resources, nil
}

func allowsList(r metav1.APIResource) bool {
	return slices.Contains(r.Verbs, "list")
}

func (e *exporter) printSummary(resources []*types.GroupResource) error {
	withPages := e.config.QueryPageSize > 0
	withErrors := e.config.Verbose && e.stats.HasErrors()

	header := []string{"Group", "Version", "Kind", "Namespaced", "Instances", "Exported"}
	aligns := []tw.Align{
		tw.AlignLeft,
		tw.AlignLeft,
		tw.AlignLeft,
		tw.AlignCenter,
		tw.AlignRight,
		tw.AlignRight,
	}
	if e.config.PrintSize {
		header = append(header, "Size")
		aligns = append(aligns, tw.AlignRight)
	}
	header = append(header, "Query Duration")
	aligns = append(aligns, tw.AlignRight)
	if withPages {
		header = append(header, "Pages")
		aligns = append(aligns, tw.AlignRight)
	}
	header = append(header, "Export Duration")
	aligns = append(aligns, tw.AlignRight)
	if withErrors {
		header = append(header, "Error")
		aligns = append(aligns, tw.AlignLeft)
	}

	alignment := tw.CellAlignment{PerColumn: aligns}
	buf := &bytes.Buffer{}
	table := render.TableTo(buf,
		// borderless table, the header and the totals are separated by a line
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off},
			Settings: tw.Settings{
				Lines:      tw.Lines{ShowHeaderLine: tw.On, ShowFooterLine: tw.On},
				Separators: tw.Separators{BetweenRows: tw.Off, BetweenColumns: tw.Off},
			},
		}),
		tablewriter.WithHeaderAlignmentConfig(alignment),
		tablewriter.WithRowAlignmentConfig(alignment),
		tablewriter.WithFooterAlignmentConfig(alignment),
	)
	table.Header(header)

	var qd, ed time.Duration
	var inst, totalInst, pages int
	var size int64

	for _, r := range resources {
		if err := table.Append(r.Report(e.config.PrintSize, withErrors, withPages)); err != nil {
			return err
		}
		qd += r.QueryDuration
		ed += r.ExportDuration
		totalInst += r.Instances
		inst += r.ExportedInstances
		size += r.ExportedSize
		pages += r.Pages
	}

	total := "TOTAL"
	if e.config.Worker > 1 {
		total = "CUMULATED " + total
	}
	footer := []string{
		total,
		"",
		"",
		"",
		strconv.Itoa(totalInst),
		strconv.Itoa(inst),
	}
	if e.config.PrintSize {
		footer = append(footer, humanize.Bytes(uint64(size)))
	}
	footer = append(footer, types.FormatDuration(qd))
	if withPages {
		footer = append(footer, strconv.Itoa(pages))
	}
	footer = append(footer, types.FormatDuration(ed))
	if withErrors {
		footer = append(footer, "")
	}
	table.Footer(footer)

	e.printHeading("📊", "Summary")
	if err := table.Render(); err != nil {
		return err
	}
	e.printIndented(buf.String())
	return nil
}

// printIndented prints the given block indented like all other content below a heading.
func (e *exporter) printIndented(block string) {
	for line := range strings.SplitSeq(strings.TrimRight(block, "\n"), "\n") {
		// the table adds a leading padding space, replace it by the common indentation
		if strings.HasPrefix(line, " ") {
			line = strings.TrimPrefix(line, " ")
		} else {
			// separator lines have no padding, shorten them to the content width
			line = strings.TrimSuffix(line, "─")
		}
		e.l.Printf("%s\n", indent+strings.TrimRight(line, " "))
	}
}

func (e *exporter) printStats() {
	e.printHeading("✅", "Result")
	printEntries(e.l, e.statsEntries())
}

func (e *exporter) statsEntries() []entry {
	var entries []entry
	if e.archive != "" {
		entries = append(entries, entry{"💾", "archive", e.archive})
		if len(e.deletedArchives) > 0 {
			entries = append(entries, entry{"🚮", "deleted archives", strconv.Itoa(len(e.deletedArchives))})
		}
	}
	entries = append(entries, entry{"📜", "kinds", strconv.Itoa(e.stats.Kinds)})
	if e.config.QueryPageSize > 0 {
		entries = append(entries, entry{"📃", "query pages", strconv.Itoa(e.stats.Pages)})
	}
	entries = append(entries, entry{"📚", "exported resources", strconv.Itoa(e.stats.Resources)})
	if e.config.PrintSize {
		entries = append(entries, entry{"📏", "exported size", humanize.Bytes(uint64(e.stats.ExportedSize))})
	}
	entries = append(entries, entry{"🏠", "namespaces", strconv.Itoa(e.stats.Namespaces())})
	if e.stats.HasErrors() {
		entries = append(entries, entry{"❗", "errors", strconv.Itoa(e.stats.Errors)})
	}
	return append(entries, entry{"⏳", "duration", types.FormatDuration(time.Since(e.start))})
}

func (e *exporter) purgeTarget() error {
	if _, err := os.Stat(e.config.Target); os.IsNotExist(err) {
		return nil
	}

	e.l.Printf("Deleting target %q\n", e.config.Target)
	defer e.l.Checkf("done 🚮\n")
	return os.RemoveAll(e.config.Target)
}

func (e *exporter) Stats() *worker.Stats {
	return e.stats
}

func (e *exporter) Config() *types.Config {
	return e.config
}

func (e *exporter) Start() time.Time {
	return e.start
}

func (e *exporter) Logger() log.YALI {
	return e.l
}

func (e *exporter) ClusterHost() string {
	if e.ac != nil && e.ac.RestConfig != nil {
		return e.ac.RestConfig.Host
	}
	return ""
}
