package types

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/bakito/kubexporter/pkg/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	// DefaultFileNameTemplate default file name template
	DefaultFileNameTemplate = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Name}}.{{.Extension}}`
	// DefaultListFileNameTemplate default list file name template
	DefaultListFileNameTemplate = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Extension}}`
	// DefaultFormat default output format
	DefaultFormat = "yaml"
	// DefaultTarget default export target dir
	DefaultTarget = "exports"

	// ProgressBar progress bar mode
	ProgressBar = Progress("bar")
	// ProgressSimple simple progress mode
	ProgressSimple = Progress("simple")
	// ProgressNone no progress
	ProgressNone = Progress("none")
)

var (
	invalidFileChars = regexp.MustCompile(`[^a-zA-Z0-9.\-]`)
	// DefaultExcludedFields the default field to be excluded
	DefaultExcludedFields = [][]string{
		{"status"},
		{"metadata", "uid"},
		{"metadata", "selfLink"},
		{"metadata", "resourceVersion"},
		{"metadata", "creationTimestamp"},
		{"metadata", "generation"},
		{"metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration"},
	}
)

// NewConfig create a new config
func NewConfig(configFlags *genericclioptions.ConfigFlags, printFlags *genericclioptions.PrintFlags) *Config {
	return &Config{
		FileNameTemplate:     DefaultFileNameTemplate,
		ListFileNameTemplate: DefaultListFileNameTemplate,
		Target:               DefaultTarget,
		Summary:              false,
		Progress:             ProgressBar,
		Worker:               1,
		Excluded: Excluded{
			Fields: DefaultExcludedFields,
		},
		configFlags: configFlags,
		printFlags:  printFlags,
	}
}

// Config export config
type Config struct {
	Excluded             Excluded `yaml:"excluded"`
	Included             Included `yaml:"included"`
	FileNameTemplate     string   `yaml:"fileNameTemplate"`
	ListFileNameTemplate string   `yaml:"listFileNameTemplate"`
	AsLists              bool     `yaml:"asLists"`
	Target               string   `yaml:"target"`
	ClearTarget          bool     `yaml:"clearTarget"`
	Summary              bool     `yaml:"summary"`
	Progress             Progress `yaml:"progress"`
	Namespace            string   `yaml:"namespace"`
	Worker               int      `yaml:"worker"`
	Archive              bool     `yaml:"archive"`
	Quiet                bool     `yaml:"quiet"`
	Verbose              bool     `yaml:"verbose"`

	excludedSet set
	includedSet set
	log         log.YALI
	configFlags *genericclioptions.ConfigFlags
	printFlags  *genericclioptions.PrintFlags
}

// Progress type
type Progress string

// Excluded exclusion params
type Excluded struct {
	Kinds      []string              `yaml:"kinds"`
	Fields     [][]string            `yaml:"fields"`
	KindFields map[string][][]string `yaml:"kindFields"`
}

// Included inclusion params
type Included struct {
	Kinds []string `yaml:"kinds"`
}

// FilterFields filter fields for a given resource
func (c *Config) FilterFields(res *GroupResource, us unstructured.Unstructured) {
	for _, f := range c.Excluded.Fields {
		removeNestedField(us.Object, f...)
	}
	gk := res.GroupKind()
	if c.Excluded.KindFields != nil && c.Excluded.KindFields[gk] != nil {
		for _, f := range c.Excluded.KindFields[gk] {
			removeNestedField(us.Object, f...)
		}
	}
}

// removeNestedField removes the nested field from the obj.
func removeNestedField(obj map[string]interface{}, fields ...string) {
	m := obj
	for i, field := range fields[:len(fields)-1] {
		if x, ok := m[field].(map[string]interface{}); ok {
			m = x
		} else {
			if x, ok := m[field].([]interface{}); ok {
				for _, y := range x {
					if yy, ok := y.(map[string]interface{}); ok {
						removeNestedField(yy, fields[i+1:]...)
					}
				}
			}
			return
		}
	}
	delete(m, fields[len(fields)-1])
}

// IsExcluded check if the group resource is excluded
func (c *Config) IsExcluded(gr *GroupResource) bool {
	if len(c.Included.Kinds) > 0 {
		if c.includedSet == nil {
			c.includedSet = newSet(c.Included.Kinds...)
		}

		return !c.includedSet.contains(gr.GroupKind())
	}

	if c.excludedSet == nil {
		c.excludedSet = newSet(c.Excluded.Kinds...)
	}

	return c.excludedSet.contains(gr.GroupKind())
}

// FileName generate export file name
func (c *Config) FileName(res *GroupResource, us *unstructured.Unstructured, index int) (string, error) {
	name := us.GetName()
	if index > 0 {
		// if index > 0 -> same name with different cases -> we add an index
		name = fmt.Sprintf("%s_%d", us.GetName(), index)
	}
	return c.fileName(res, us.GetNamespace(), name, c.FileNameTemplate)
}

// ListFileName generate export list file name
func (c *Config) ListFileName(res *GroupResource, namespace string) (string, error) {
	return c.fileName(res, namespace, "", c.ListFileNameTemplate)
}

func (c *Config) fileName(res *GroupResource, namespace string, name string, templ string) (string, error) {
	t, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(templ)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, map[string]string{
		"Namespace": namespace,
		"Name":      name,
		"Kind":      res.Kind(),
		"Group":     res.APIGroup,
		"Extension": *c.printFlags.OutputFormat},
	)

	path := tpl.String()
	var pathElements []string
	for _, e := range strings.Split(filepath.Clean(filepath.Join(path)), string(os.PathSeparator)) {
		pathElements = append(pathElements, invalidFileChars.ReplaceAllString(e, "_"))
	}
	return filepath.Join(pathElements...), err
}

// Validate validate the config
func (c *Config) Validate() error {
	if c.FileNameTemplate == "" {
		return fmt.Errorf("file name template must not be empty")
	} else if _, err := c.FileName(&GroupResource{}, &unstructured.Unstructured{}, 0); err != nil {
		return fmt.Errorf("error parsing file name template [%s]", c.FileNameTemplate)
	}
	if c.ListFileNameTemplate == "" {
		return fmt.Errorf("list file name template must not be empty")
	} else if _, err := c.ListFileName(&GroupResource{}, ""); err != nil {
		return fmt.Errorf("error parsing list file name template [%s]", c.ListFileNameTemplate)
	}
	if c.Worker <= 0 {
		return fmt.Errorf("worker must be > 0")
	}

	abs, err := filepath.Abs(c.Target)
	if err != nil {
		return err
	}
	c.Target = abs

	if c.Quiet {
		c.Summary = false
		c.Progress = ProgressNone
	}

	if c.Progress != ProgressSimple && c.Progress != ProgressNone {
		c.Progress = ProgressBar
	}
	return nil
}

// PrintObj print the given object
func (c *Config) PrintObj(ro runtime.Object, out io.Writer) error {
	p, err := c.printFlags.ToPrinter()
	if err != nil {
		return err
	}
	return p.PrintObj(ro.(runtime.Object), out)
}

// Logger get the logger
func (c *Config) Logger() log.YALI {
	if c.log == nil {
		c.log = log.New(c.Quiet, c.Progress == ProgressSimple)
	}
	return c.log
}

// OutputFormat get the current output format
func (c *Config) OutputFormat() string {
	if c.printFlags != nil && c.printFlags.OutputFormat != nil {
		return *c.printFlags.OutputFormat
	}
	return ""
}

// RestConfig get the current rest config
func (c *Config) RestConfig() (*rest.Config, error) {
	// try to find a cube config
	cfg, err := cmdutil.NewFactory(c.configFlags).ToRESTConfig()
	if err == nil {
		return cfg, nil
	}

	// try in cluster config
	return rest.InClusterConfig()
}

type set map[string]bool

func (s set) contains(key string) bool {
	_, ok := s[key]
	return ok
}

func (s set) add(key string) {
	s[key] = true
}

func newSet(values ...string) set {
	s := make(set)
	for _, v := range values {
		s.add(v)
	}
	return s
}
