package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	invalidFileChars = regexp.MustCompile(`[^a-zA-Z0-9.\-]`)
)

// GroupResource group resource information
type GroupResource struct {
	APIGroup        string
	APIGroupVersion string
	APIResource     metav1.APIResource
	APIVersion      string
	Instances       int
	Error           string
	QueryDuration   time.Duration
	ExportDuration  time.Duration
}

// Report generate report rows
func (r GroupResource) Report(withError bool) []string {
	row := []string{
		r.APIGroup,
		r.APIVersion,
		r.APIResource.Kind,
		strconv.FormatBool(r.APIResource.Namespaced),
		strconv.Itoa(r.Instances),
		r.QueryDuration.String(),
		r.ExportDuration.String(),
	}
	if withError {
		row = append(row, r.Error)
	}
	return row
}

// GroupKind get concatenated group and kind
func (r GroupResource) GroupKind() string {
	if r.APIGroup != "" {
		return fmt.Sprintf("%s.%s", r.APIGroup, r.APIResource.Kind)
	}
	return r.APIResource.Kind
}

// Sort sort GroupResource
func Sort(resources []*GroupResource) func(int, int) bool {
	return func(i, j int) bool {
		ret := strings.Compare(resources[i].APIGroup, resources[j].APIGroup)
		if ret > 0 {
			return false
		} else if ret == 0 {
			return strings.Compare(resources[i].APIResource.Kind, resources[j].APIResource.Kind) < 0
		}
		return true
	}
}

// Config export config
type Config struct {
	Excluded         Excluded `yaml:"excluded"`
	excludedSet      set
	FileNameTemplate string `yaml:"fileNameTemplate"`
	OutputFormat     string `yaml:"outputFormat"`
	Target           string `yaml:"target"`
	Summary          bool   `yaml:"summary"`
	Progress         bool   `yaml:"progress"`
	Namespace        string `yaml:"namespace"`
	Worker           int    `yaml:"worker"`
	Archive          bool   `yaml:"archive"`
}

// Excluded exclusion params
type Excluded struct {
	Kinds      []string              `yaml:"kinds"`
	Fields     [][]string            `yaml:"fields"`
	KindFields map[string][][]string `yaml:"kindFields"`
}

// IsExcluded check if the group resource is excluded
func (c *Config) IsExcluded(gr *GroupResource) bool {
	if c.excludedSet == nil {
		c.excludedSet = newSet(c.Excluded.Kinds...)
	}
	return c.excludedSet.contains(gr.GroupKind())
}

// FileName generate export file name
func (c *Config) FileName(res *GroupResource, us *unstructured.Unstructured) (string, error) {
	t, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(c.FileNameTemplate)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, map[string]string{
		"Namespace": us.GetNamespace(),
		"Name":      us.GetName(),
		"Kind":      us.GetKind(),
		"Group":     res.APIGroup,
		"Extension": c.OutputFormat},
	)

	path := tpl.String()

	pathElements := []string{invalidFileChars.ReplaceAllString(c.Target, "_")}
	for _, e := range filepath.SplitList(filepath.Dir(path)) {
		pathElements = append(pathElements, invalidFileChars.ReplaceAllString(e, "_"))
	}
	pathElements = append(pathElements, invalidFileChars.ReplaceAllString(filepath.Base(path), "_"))
	return filepath.Join(pathElements...), err
}

// Validate validate the config
func (c *Config) Validate() error {
	if c.OutputFormat != "json" && c.OutputFormat != "yaml" {
		return fmt.Errorf("unsupported output format [%s]", c.OutputFormat)
	}
	if _, err := c.FileName(&GroupResource{}, &unstructured.Unstructured{}); err != nil {
		return fmt.Errorf("error parsing template [%s]", c.FileNameTemplate)
	}
	if c.Worker <= 0 {
		return fmt.Errorf("worker must be > 0")
	}

	return nil
}

// Marshal marshal the unstructured with the correct format
func (c *Config) Marshal(us *unstructured.Unstructured) ([]byte, error) {
	switch c.OutputFormat {
	case "yaml":
		return yaml.Marshal(us)
	case "json":
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		enc.SetIndent("", "  ")
		err := enc.Encode(us)
		return out.Bytes(), err
	}
	return nil, fmt.Errorf("unsupported output format [%s]", c.OutputFormat)

}

// FilterFields filter fields for a given resource
func (e *Excluded) FilterFields(res *GroupResource, us unstructured.Unstructured) {
	for _, f := range e.Fields {
		unstructured.RemoveNestedField(us.Object, f...)
	}
	if e.KindFields != nil && e.KindFields[res.GroupKind()] != nil {
		for _, f := range e.KindFields[res.GroupKind()] {
			unstructured.RemoveNestedField(us.Object, f...)
		}
	}
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
