package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
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

type groupResource struct {
	APIGroup        string
	APIGroupVersion string
	APIResource     metav1.APIResource
	APIVersion      string
	Instances       int
	Error           string
	QueryDuration   time.Duration
	ExportDuration  time.Duration
}

func (r groupResource) Report(withError bool) []string {
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

func (r groupResource) GroupKind() string {
	if r.APIGroup != "" {
		return fmt.Sprintf("%s.%s", r.APIGroup, r.APIResource.Kind)
	}
	return r.APIResource.Kind
}

type Config struct {
	Excluded         Excluded `yaml:"excluded"`
	excludedSet      set
	FileNameTemplate string `yaml:"fileNameTemplate"`
	OutputFormat     string `yaml:"format"`
	Target           string `yaml:"target"`
	Summary          bool   `yaml:"summary"`
	Progress         bool   `yaml:"progress"`
	Zip              bool   `yaml:"zip"`
	Namespace        string `yaml:"namespace"`
}

type Excluded struct {
	Kinds      []string              `yaml:"kinds"`
	Fields     [][]string            `yaml:"fields"`
	KindFields map[string][][]string `yaml:"kindFields"`
}

func (c *Config) IsExcluded(gr groupResource) bool {
	if c.excludedSet == nil {
		c.excludedSet = newSet(c.Excluded.Kinds...)
	}
	return c.excludedSet.contains(gr.GroupKind())
}

func (c *Config) FileName(res groupResource, us *unstructured.Unstructured) (string, error) {
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

func (c *Config) Validate() error {
	if c.OutputFormat != "json" && c.OutputFormat != "yaml" {
		return fmt.Errorf("unsupported output format [%s]", c.OutputFormat)
	}
	if _, err := c.FileName(groupResource{}, &unstructured.Unstructured{}); err != nil {
		return fmt.Errorf("error parsing template [%s]", c.FileNameTemplate)
	}

	return nil
}

func (c *Config) Marshal(us *unstructured.Unstructured) ([]byte, error) {
	switch c.OutputFormat {
	case "yaml":
		return yaml.Marshal(us)
	case "json":
		return json.Marshal(us)
	}
	return nil, fmt.Errorf("unsupported output format [%s]", c.OutputFormat)

}

func (e *Excluded) FilterFields(res groupResource, us unstructured.Unstructured) {
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
