package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/bakito/kubexporter/pkg/log"
	"github.com/ghodss/yaml"
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

	// DefaultMaskReplacement Default Mask Replacement
	DefaultMaskReplacement = "*****"
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

// Update the config from the file with given path
func UpdateFrom(config *Config, path string) error {

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, config)
}

// NewConfig create a new config
func NewConfig(configFlags *genericclioptions.ConfigFlags, printFlags *genericclioptions.PrintFlags) *Config {
	return &Config{
		FileNameTemplate:     DefaultFileNameTemplate,
		ListFileNameTemplate: DefaultListFileNameTemplate,
		Target:               DefaultTarget,
		Summary:              false,
		Progress:             ProgressBar,
		Worker:               1,
		Masked: Masked{
			KindFields: KindFields{},
		},
		Excluded: Excluded{
			Fields:       DefaultExcludedFields,
			KindFields:   KindFields{},
			KindsByField: make(map[string][]FieldValue),
		},
		SortSlices:  KindFields{},
		configFlags: configFlags,
		printFlags:  printFlags,
	}
}

// Config export config
type Config struct {
	Excluded             Excluded   `json:"excluded" yaml:"excluded"`
	Included             Included   `json:"included" yaml:"included"`
	Masked               Masked     `json:"masked" yaml:"masked"`
	SortSlices           KindFields `json:"sortSlices" yaml:"sortSlices"`
	FileNameTemplate     string     `json:"fileNameTemplate" yaml:"fileNameTemplate"`
	ListFileNameTemplate string     `json:"listFileNameTemplate" yaml:"listFileNameTemplate"`
	AsLists              bool       `json:"asLists" yaml:"asLists"`
	Target               string     `json:"target" yaml:"target"`
	ClearTarget          bool       `json:"clearTarget" yaml:"clearTarget"`
	Summary              bool       `json:"summary" yaml:"summary"`
	Progress             Progress   `json:"progress" yaml:"progress"`
	Namespace            string     `json:"namespace" yaml:"namespace"`
	Worker               int        `json:"worker" yaml:"worker"`
	Archive              bool       `json:"archive" yaml:"archive"`
	Quiet                bool       `json:"quiet" yaml:"quiet"`
	Verbose              bool       `json:"verbose" yaml:"verbose"`

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
	Kinds        []string                `json:"kinds" yaml:"kinds"`
	Fields       [][]string              `json:"fields" yaml:"fields"`
	KindFields   KindFields              `json:"kindFields" yaml:"kindFields"`
	KindsByField map[string][]FieldValue `json:"kindByField" yaml:"kindByField"`
}

// Masked masking params
type Masked struct {
	Replacement string     `json:"replacement" yaml:"replacement"`
	KindFields  KindFields `json:"kindFields" yaml:"kindFields"`
}

// KindFields map kinds to fields
type KindFields map[string][][]string

// Included inclusion params
type Included struct {
	Kinds []string `json:"kinds" yaml:"kinds"`
}

// FieldValue field with value
type FieldValue struct {
	Field  []string `json:"field" yaml:"field"`
	Values []string `json:"values" yaml:"values"   `
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

// MaskFields mask fields for a given resource
func (c *Config) MaskFields(res *GroupResource, us unstructured.Unstructured) {
	gk := res.GroupKind()
	if c.Masked.KindFields != nil && c.Masked.KindFields[gk] != nil {
		for _, f := range c.Masked.KindFields[gk] {
			maskNestedField(us.Object, c.Masked.Replacement, f...)
		}
	}
}

// maskNestedField masks the nested field from the obj.
func maskNestedField(obj map[string]interface{}, rep string, fields ...string) {
	m := obj
	for i, field := range fields[:len(fields)-1] {
		if x, ok := m[field].(map[string]interface{}); ok {
			m = x
		} else {
			if x, ok := m[field].([]interface{}); ok {
				for _, y := range x {
					if yy, ok := y.(map[string]interface{}); ok {
						maskNestedField(yy, rep, fields[i+1:]...)
					}
				}
			}
			return
		}
	}
	switch e := m[fields[len(fields)-1]].(type) {
	case map[string]interface{}:
		for k := range e {
			e[k] = rep
		}
	case string:
		m[fields[len(fields)-1]] = rep
	default:
		println()
	}
}

// SortSliceFields sort fields for a given resource
func (c *Config) SortSliceFields(res *GroupResource, us unstructured.Unstructured) {
	gk := res.GroupKind()
	if c.SortSlices != nil && c.SortSlices[gk] != nil {
		for _, f := range c.SortSlices[gk] {
			if sl, ok, err := unstructured.NestedSlice(us.Object, f...); ok && err == nil {
				if len(sl) > 0 {
					switch sl[0].(type) {
					case string:
						sort.Slice(sl, func(i, j int) bool {
							return sl[i].(string) < sl[j].(string)
						})
					case int64:
						sort.Slice(sl, func(i, j int) bool {
							return sl[i].(int64) < sl[j].(int64)
						})
					case float64:
						sort.Slice(sl, func(i, j int) bool {
							return sl[i].(float64) < sl[j].(float64)
						})
					default:
						sort.Slice(sl, func(i, j int) bool {
							a, _ := json.Marshal(sl[i])
							b, _ := json.Marshal(sl[i])
							return string(a) < string(b)
						})
					}
					_ = unstructured.SetNestedSlice(us.Object, sl, f...)
				}
			}
		}
	}
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

// IsInstanceExcluded check if the kind instance is excluded
func (c *Config) IsInstanceExcluded(res *GroupResource, us unstructured.Unstructured) bool {
	if fvs, ok := c.Excluded.KindsByField[res.GroupKind()]; ok {
		for _, fv := range fvs {
			for _, v := range fv.Values {
				if matches(us, fv.Field, v) {
					return true
				}
			}
		}
	}
	return false
}

func matches(us unstructured.Unstructured, field []string, filter string) bool {
	if v, ok, err := unstructured.NestedFieldCopy(us.Object, field...); ok && err == nil && v != nil {
		value := fmt.Sprintf("%v", v)
		if value == filter {
			return true
		}
	}
	return false
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
