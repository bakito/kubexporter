package export

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bakito/kubexporter/pkg/client"
	"github.com/bakito/kubexporter/pkg/export/worker"
	"github.com/bakito/kubexporter/pkg/log"
	"github.com/bakito/kubexporter/pkg/render"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/version"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewExporter create a new exporter
func NewExporter(config *types.Config) (Exporter, error) {
	ac, err := client.NewApiClient(config)
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

// Exporter interface
type Exporter interface {
	Export() error
}

func (e *exporter) Export() error {
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

	sort.SliceStable(resources, types.Sort(resources))

	var prog *mpb.Progress

	var mainBar *mpb.Bar
	if e.config.Progress == types.ProgressBar {
		prog = mpb.New()
		mainBar = prog.AddBar(int64(len(resources)),
			mpb.PrependDecorators(
				// display our name with one space on the right
				decor.Name("Resources", decor.WC{W: len("Resources") + 1, C: decor.DindentRight}),
				decor.Elapsed(decor.ET_STYLE_GO),
			),
			mpb.AppendDecorators(
				decor.CurrentNoUnit(""),
				decor.Name("/"),
				decor.TotalNoUnit(""),
				decor.Name(" "),
				decor.Percentage(),
			),
		)
	}

	var workers []worker.Worker
	for i := 0; i < e.config.Worker; i++ {
		workers = append(workers, worker.New(i, e.config, e.ac, prog, mainBar))
	}

	s, err := worker.RunExport(workers, resources)
	if err != nil {
		return err
	}
	e.stats.Add(s)

	if prog != nil {
		prog.Wait()
	}

	if e.config.Summary {
		e.printSummary(resources)
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
	}
	return nil
}

func (e *exporter) writeIntro() {
	e.l.Printf("Starting export ...\n")
	e.l.Printf("  kubexporter version %q\n", version.Version)
	e.l.Printf("  cluster %q\n", e.ac.RestConfig.Host)
	if e.config.Namespace == "" {
		e.l.Printf("  all namespaces 🏘️\n")
	} else {
		e.l.Printf("  namespace %q 🏠\n", e.config.Namespace)
	}
	e.l.Printf("  target %q 📁\n", e.config.Target)
	e.l.Printf("  format %q 📜\n", e.config.OutputFormat())
	if e.config.Worker > 1 {
		if e.config.Progress == types.ProgressBar {
			e.l.Printf("  worker %s\n", strings.Repeat("👷‍️", e.config.Worker))
		} else {
			e.l.Printf("  worker %d\n", e.config.Worker)
		}
	}
	if e.config.Summary {
		e.l.Printf("  summary 📊\n")
	}
	if e.config.ConsiderOwnerReferences {
		e.l.Printf("  considering owner references 👑\n")
	}

	if len(e.config.Masked.KindFields) > 0 {
		e.l.Printf("  masked fields 🤿 %v\n", e.config.Masked.KindFields)
	}
	if len(e.config.Encrypted.KindFields) > 0 {
		e.l.Printf("  encrypted fields 🔒 %v\n", e.config.Encrypted.KindFields)
	}

	if e.config.AsLists {
		e.l.Printf("  as lists 📦\n")
	} else if e.config.QueryPageSize != 0 {
		e.l.Printf("  query page size %d 📃\n", e.config.QueryPageSize)
	}
	if e.config.Archive {
		e.l.Printf("  compress as archive ️🗜\n")
		if e.config.ArchiveRetentionDays > 0 {
			e.l.Printf("  delete archives older than %d days 🚮\n", e.config.ArchiveRetentionDays)
		}
	}
	e.config.Logger().Printf("\nExporting ...\n")
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
			if !allowsList(resource) || e.config.IsExcluded(r) || (!resource.Namespaced && e.config.Namespace != "") {
				continue
			}

			resources = append(resources, r)
		}
	}
	return resources, nil
}

func allowsList(r metav1.APIResource) bool {
	for _, v := range r.Verbs {
		if v == "list" {
			return true
		}
	}
	return false
}

func (e *exporter) printSummary(resources []*types.GroupResource) {
	withPages := e.config.QueryPageSize > 0

	table := render.Table()
	header := []string{"Group", "Version", "Kind", "Namespaces", "Total Instances", "Exported Instances", "Query Duration"}
	if withPages {
		header = append(header, "Query Pages")
	}
	header = append(header, "Export Duration")
	if e.config.Verbose && e.stats.HasErrors() {
		header = append(header, "Error")
	}
	table.SetHeader(header)
	start := time.Now()
	qd := start
	ed := start
	var inst int
	var totalInst int
	var pages int

	for _, r := range resources {
		table.Append(r.Report(e.config.Verbose && e.stats.HasErrors(), withPages))
		qd = qd.Add(r.QueryDuration)
		ed = ed.Add(r.ExportDuration)
		totalInst += r.Instances
		inst += r.ExportedInstances
		pages += r.Pages
	}
	total := "TOTAL"
	if e.config.Worker > 1 {
		total = "CUMULATED " + total
	}
	totalRow := []string{total, "", "", "", strconv.Itoa(inst), strconv.Itoa(totalInst), qd.Sub(start).String()}
	if withPages {
		totalRow = append(totalRow, strconv.Itoa(pages))
	}
	totalRow = append(totalRow, ed.Sub(start).String())
	table.Append(totalRow)
	table.Render()
}

func (e *exporter) printStats() {
	println()
	if e.archive != "" {
		e.l.Checkf("🗜\tArchive %s\n", e.archive)
		if len(e.deletedArchives) > 0 {
			e.l.Checkf("🚮\tDeleted old Archives %d\n", len(e.deletedArchives))
		}
	}
	e.l.Checkf("📜\tKinds %d\n", e.stats.Kinds)
	if e.config.QueryPageSize > 0 {
		e.l.Checkf("📃\tQuery Pages %d\n", e.stats.Pages)
	}
	e.l.Checkf("🗃\tExported Resources %d\n", e.stats.Resources)
	e.l.Checkf("🏠\tNamespaces %d\n", e.stats.Namespaces())
	if e.stats.HasErrors() {
		e.l.Checkf("⚠️\tErrors %d\n", e.stats.Errors)
	}
	e.l.Checkf("⏱️\tDuration %s\n", time.Since(e.start).String())
}

func (e *exporter) purgeTarget() error {
	if _, err := os.Stat(e.config.Target); os.IsNotExist(err) {
		return nil
	}

	e.l.Printf("Deleting target %q\n", e.config.Target)
	defer e.l.Checkf("done 🚮\n")
	return os.RemoveAll(e.config.Target)
}

type exporter struct {
	start           time.Time
	l               log.YALI
	config          *types.Config
	stats           *worker.Stats
	archive         string
	deletedArchives []string
	ac              *client.ApiClient
}
