package export

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bakito/kubexporter/pkg/export/worker"
	"github.com/bakito/kubexporter/pkg/log"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/olekukonko/tablewriter"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// NewExporter create a new exporter
func NewExporter(config *types.Config) (Exporter, error) {
	err := config.Validate()
	if err != nil {
		return nil, err
	}

	rc, err := config.RestConfig()
	if err != nil {
		return nil, err
	}

	return &exporter{
		config:     config,
		restConfig: rc,
		l:          config.Logger(),
		stats:      &worker.Stats{},
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

	dcl, err := discovery.NewDiscoveryClientForConfig(e.restConfig)
	if err != nil {
		return err
	}

	resources, err := e.listResources(dcl)
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
				decor.Name("Resources", decor.WC{W: len("Resources") + 1, C: decor.DidentRight}),
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

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dcl))
	client, err := dynamic.NewForConfig(e.restConfig)
	if err != nil {
		return err
	}

	var workers []worker.Worker
	for i := 0; i < e.config.Worker; i++ {
		workers = append(workers, worker.New(i, e.config, mapper, client, prog, mainBar))
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
	}
	return err
}

func (e *exporter) writeIntro() {
	e.l.Printf("Starting export ...\n")
	e.l.Printf("  cluster %q\n", e.restConfig.Host)
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
	if e.config.AsLists {
		e.l.Printf("  as lists 📦\n")
	}
	if e.config.Archive {
		e.l.Printf("  compress as archive 🗜️\n")
	}
	e.config.Logger().Printf("\nExporting ...\n")
}

func (e *exporter) listResources(dcl *discovery.DiscoveryClient) ([]*types.GroupResource, error) {
	lists, err := dcl.ServerPreferredResources()
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
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeaderLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	header := []string{"Group", "Version", "Kind", "Namespaces", "Instances", "Query Duration", "Export Duration"}
	if e.config.Verbose && e.stats.HasErrors() {
		header = append(header, "Error")
	}
	table.SetHeader(header)
	start := time.Now()
	qd := start
	ed := start
	var inst int

	for _, r := range resources {
		table.Append(r.Report(e.config.Verbose && e.stats.HasErrors()))
		qd = qd.Add(r.QueryDuration)
		ed = ed.Add(r.ExportDuration)
		inst += r.Instances
	}
	total := "TOTAL"
	if e.config.Worker > 1 {
		total = "CUMULATED " + total
	}
	table.Append([]string{total, "", "", "", strconv.Itoa(inst), qd.Sub(start).String(), ed.Sub(start).String()})
	table.Render()
}

func (e *exporter) printStats() {
	if e.archive != "" {
		e.l.Checkf("Archive    🗜️  %s\n", e.archive)
	}
	e.l.Checkf("Kinds      📜%12d\n", e.stats.Kinds)
	e.l.Checkf("Resources  🗃 ️%12d\n", e.stats.Resources)
	e.l.Checkf("Namespaces 🏘️ %12d\n", e.stats.Namespaces())
	if e.stats.HasErrors() {
		e.l.Checkf("Errors     ⚠️ %12d\n", e.stats.Errors)
	}
	e.l.Checkf("Duration   ⌛ %s\n", time.Since(e.start).String())
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
	start      time.Time
	l          log.YALI
	config     *types.Config
	restConfig *rest.Config
	stats      *worker.Stats
	archive    string
}
