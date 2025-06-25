package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/bakito/kubexporter/pkg/log"
)

const (
	// DefaultFileNameTemplate default file name template.
	DefaultFileNameTemplate = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Name}}.{{.Extension}}`
	// DefaultListFileNameTemplate default list file name template.
	DefaultListFileNameTemplate = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Extension}}`
	// DefaultFormat default output format.
	DefaultFormat = "yaml"
	// DefaultTarget default export target dir.
	DefaultTarget = "exports"

	// ProgressBar progress bar mode.
	ProgressBar = Progress("bar")
	// ProgressBarBubbles bubbles progress bar mode.
	ProgressBarBubbles = Progress("bubbles")
	// ProgressSimple simple progress mode.
	ProgressSimple = Progress("simple")
	// ProgressNone no progress.
	ProgressNone = Progress("none")

	// DefaultMaskReplacement Default Mask Replacement.
	DefaultMaskReplacement = "*****"
)

var (
	invalidFileChars = regexp.MustCompile(`[^a-zA-Z0-9.\-]`)
	// DefaultExcludedFields the default field to be excluded.
	DefaultExcludedFields = [][]string{
		{"status"},
		{"metadata", "uid"},
		{"metadata", "selfLink"},
		{"metadata", "resourceVersion"},
		{"metadata", "creationTimestamp"},
		{"metadata", "deletionTimestamp"},
		{"metadata", "deletionGracePeriodSeconds"},
		{"metadata", "generation"},
		{"metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration"},
	}
)

// UpdateFrom the config from the file with given path.
func UpdateFrom(config *Config, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, config)
}

// NewConfig create a new config.
func NewConfig(configFlags *genericclioptions.ConfigFlags, printFlags *genericclioptions.PrintFlags) *Config {
	return &Config{
		FileNameTemplate:     DefaultFileNameTemplate,
		ListFileNameTemplate: DefaultListFileNameTemplate,
		QueryPageSize:        0,
		Target:               DefaultTarget,
		Summary:              false,
		Progress:             ProgressBar,
		Worker:               1,
		Masked: &Masked{
			KindFields: KindFields{},
		},
		Encrypted: &Encrypted{
			KindFields: KindFields{},
		},
		Excluded: Excluded{
			Fields:       DefaultExcludedFields,
			KindFields:   KindFields{},
			KindsByField: make(map[string][]FieldValue),
		},
		SortSlices:  KindFields{},
		configFlags: configFlags,
		PrintFlags:  printFlags,
	}
}

// Config export config.
type Config struct {
	Excluded                Excluded      `json:"excluded"                yaml:"excluded"`
	Included                Included      `json:"included"                yaml:"included"`
	CreatedWithin           time.Duration `json:"createdWithin"           yaml:"createdWithin"`
	ConsiderOwnerReferences bool          `json:"considerOwnerReferences" yaml:"considerOwnerReferences"`
	Masked                  *Masked       `json:"masked"                  yaml:"masked"`
	Encrypted               *Encrypted    `json:"encrypted"               yaml:"encrypted"`
	SortSlices              KindFields    `json:"sortSlices"              yaml:"sortSlices"`
	FileNameTemplate        string        `json:"fileNameTemplate"        yaml:"fileNameTemplate"`
	ListFileNameTemplate    string        `json:"listFileNameTemplate"    yaml:"listFileNameTemplate"`
	AsLists                 bool          `json:"asLists"                 yaml:"asLists"`
	QueryPageSize           int           `json:"queryPageSize"           yaml:"queryPageSize"`
	Target                  string        `json:"target"                  yaml:"target"`
	ClearTarget             bool          `json:"clearTarget"             yaml:"clearTarget"`
	Summary                 bool          `json:"summary"                 yaml:"summary"`
	Progress                Progress      `json:"progress"                yaml:"progress"`
	Namespace               string        `json:"namespace"               yaml:"namespace"`
	Worker                  int           `json:"worker"                  yaml:"worker"`
	Archive                 bool          `json:"archive"                 yaml:"archive"`
	ArchiveRetentionDays    int           `json:"archiveRetentionDays"    yaml:"archiveRetentionDays"`
	ArchiveTarget           string        `json:"archiveTarget"           yaml:"archiveTarget"`
	S3Config                *S3Config     `json:"s3"                      yaml:"s3"`
	Quiet                   bool          `json:"quiet"                   yaml:"quiet"`
	Verbose                 bool          `json:"verbose"                 yaml:"verbose"`
	PrintSize               bool          `json:"printSize"               yaml:"printSize"`

	excludedSet set
	includedSet set
	log         log.YALI
	configFlags *genericclioptions.ConfigFlags
	PrintFlags  *genericclioptions.PrintFlags `json:"-" yaml:"-"`
}

func (c *Config) MaxArchiveAge() time.Time {
	return time.Now().AddDate(0, 0, -c.ArchiveRetentionDays)
}

type S3Config struct {
	Endpoint        string `json:"endpoint"        yaml:"endpoint"`
	AccessKeyID     string `json:"accessKeyID"     yaml:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey" yaml:"secretAccessKey"`
	Token           string `json:"token"           yaml:"token"`
	Secure          bool   `json:"secure"          yaml:"secure"`
	Bucket          string `json:"bucket"          yaml:"bucket"`
}

// Progress type.
type Progress string

// Excluded exclusion params.
type Excluded struct {
	Kinds        []string                `json:"kinds"       yaml:"kinds"`
	Fields       [][]string              `json:"fields"      yaml:"fields"`
	KindFields   KindFields              `json:"kindFields"  yaml:"kindFields"`
	KindsByField map[string][]FieldValue `json:"kindByField" yaml:"kindByField"`
}

// KindFields map kinds to fields.
type KindFields map[string][][]string

// Diff returns new KindFields with values that are only in the provided argument and not in this.
func (f KindFields) Diff(other KindFields) KindFields {
	diff := KindFields{}
	for thisKind, thisFields := range f {
		if otherFields, ok := other[thisKind]; ok {
			df := diffFields(thisFields, otherFields)
			if len(df) > 0 {
				diff[thisKind] = df
			}
			delete(other, thisKind)
		}
	}

	for kind, fields := range other {
		if len(fields) > 0 {
			diff[kind] = fields
		}
	}

	return diff
}

func (f KindFields) String() string {
	var kinds []string
	for k, v := range f {
		kinds = append(kinds, fmt.Sprintf("%s: [%s]", k, strings.Join(joinAll(v), ", ")))
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}

func joinAll(in [][]string) []string {
	var s []string
	for _, val := range in {
		s = append(s, fmt.Sprintf("[%s]", strings.Join(val, ",")))
	}
	return s
}

func diffFields(this, other [][]string) [][]string {
	removes := make(map[string]bool)

	for _, f := range this {
		fs := strings.Join(f, ";")
		for _, o := range other {
			myOS := strings.Join(o, ";")
			if strings.HasPrefix(myOS, fs) {
				removes[myOS] = true
			}
		}
	}

	var diff [][]string
	for _, o := range other {
		parts := strings.Join(o, ";")
		if _, ok := removes[parts]; !ok {
			diff = append(diff, o)
		}
	}

	return diff
}

// Included inclusion params.
type Included struct {
	Kinds []string `json:"kinds" yaml:"kinds"`
}

// FieldValue field with value.
type FieldValue struct {
	Field  []string `json:"field"  yaml:"field"`
	Values []string `json:"values" yaml:"values"`
}

// FilterFields filter fields for a given resource.
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
func removeNestedField(obj map[string]any, fields ...string) {
	m := obj
	for i, field := range fields[:len(fields)-1] {
		var x map[string]any
		var ok bool
		if x, ok = m[field].(map[string]any); !ok {
			if x, ok := m[field].([]any); ok {
				for _, y := range x {
					if yy, ok := y.(map[string]any); ok {
						removeNestedField(yy, fields[i+1:]...)
					}
				}
			}
			return
		}

		m = x
	}
	delete(m, fields[len(fields)-1])
}

func transformNestedFields(
	kf KindFields,
	transform func(val any) string,
	gk string,
	us unstructured.Unstructured,
) {
	if kf != nil && kf[gk] != nil {
		for _, f := range kf[gk] {
			transformNestedField(us.Object, transform, f...)
		}
	}
}

// transformNestedField transforms the nested field from the obj.
func transformNestedField(obj map[string]any, transform func(val any) string, fields ...string) {
	m := obj
	for i, field := range fields[:len(fields)-1] {
		var x map[string]any
		var ok bool
		if x, ok = m[field].(map[string]any); !ok {
			if x, ok := m[field].([]any); ok {
				for _, y := range x {
					if yy, ok := y.(map[string]any); ok {
						transformNestedField(yy, transform, fields[i+1:]...)
					}
				}
			}
			return
		}

		m = x
	}
	switch e := m[fields[len(fields)-1]].(type) {
	case map[string]any:
		for k := range e {
			e[k] = transform(e[k])
		}
	case string:
		m[fields[len(fields)-1]] = transform(m[fields[len(fields)-1]])
	}
}

// SortSliceFields sort fields for a given resource.
func (c *Config) SortSliceFields(res *GroupResource, us unstructured.Unstructured) {
	gk := res.GroupKind()
	if c.SortSlices != nil && c.SortSlices[gk] != nil {
		for _, f := range c.SortSlices[gk] {
			if sl, ok, err := unstructured.NestedSlice(us.Object, f...); ok && err == nil {
				if len(sl) > 0 {
					switch sl[0].(type) {
					case string:
						sort.Slice(sl, func(i, j int) bool {
							//nolint:forcetypeassert
							return sl[i].(string) < sl[j].(string)
						})
					case int64:
						sort.Slice(sl, func(i, j int) bool {
							//nolint:forcetypeassert
							return sl[i].(int64) < sl[j].(int64)
						})
					case float64:
						sort.Slice(sl, func(i, j int) bool {
							//nolint:forcetypeassert
							return sl[i].(float64) < sl[j].(float64)
						})
					default:
						sort.Slice(sl, func(i, j int) bool {
							a, _ := json.Marshal(sl[i])
							b, _ := json.Marshal(sl[j])
							return string(a) < string(b)
						})
					}
					_ = unstructured.SetNestedSlice(us.Object, sl, f...)
				}
			}
		}
	}
}

// IsExcluded check if the group resource is excluded.
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

// IsInstanceExcluded check if the kind instance is excluded.
func (c *Config) IsInstanceExcluded(res *GroupResource, us unstructured.Unstructured) bool {
	if c.isExcludedByOwnerReference(us) {
		return true
	}
	if c.CreatedWithin > 0 && us.GetCreationTimestamp().Time.Before(time.Now().Add(-c.CreatedWithin)) {
		return true
	}
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

func (c *Config) isExcludedByOwnerReference(us unstructured.Unstructured) bool {
	if c.ConsiderOwnerReferences && len(us.GetOwnerReferences()) > 0 {
		for _, or := range us.GetOwnerReferences() {
			gv, err := schema.ParseGroupVersion(or.APIVersion)
			r := &GroupResource{
				APIGroup:        gv.Group,
				APIVersion:      gv.Version,
				APIGroupVersion: gv.String(),
				APIResource:     metav1.APIResource{Kind: or.Kind},
			}

			if err == nil && c.IsExcluded(r) {
				return true
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

// FileName generate export file name.
func (c *Config) FileName(res *GroupResource, us *unstructured.Unstructured, index int) (string, error) {
	name := us.GetName()
	if index > 0 {
		// if index > 0 -> same name with different cases -> we add an index
		name = fmt.Sprintf("%s_%d", us.GetName(), index)
	}
	return c.fileNameInternal(res, us.GetNamespace(), name, c.FileNameTemplate)
}

// ListFileName generate export list file name.
func (c *Config) ListFileName(res *GroupResource, namespace string) (string, error) {
	return c.fileNameInternal(res, namespace, "", c.ListFileNameTemplate)
}

func (c *Config) fileNameInternal(res *GroupResource, namespace, name, templ string) (string, error) {
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
		"Extension": *c.PrintFlags.OutputFormat,
	},
	)

	path := tpl.String()
	var pathElements []string
	for _, e := range strings.Split(filepath.Clean(path), string(os.PathSeparator)) {
		pathElements = append(pathElements, invalidFileChars.ReplaceAllString(e, "_"))
	}
	return filepath.Join(pathElements...), err
}

// Validate validate the config.
func (c *Config) Validate() error {
	if c.FileNameTemplate == "" {
		return errors.New("file name template must not be empty")
	} else if _, err := c.FileName(&GroupResource{}, &unstructured.Unstructured{}, 0); err != nil {
		return fmt.Errorf("error parsing file name template [%s]", c.FileNameTemplate)
	}
	if c.ListFileNameTemplate == "" {
		return errors.New("list file name template must not be empty")
	} else if _, err := c.ListFileName(&GroupResource{}, ""); err != nil {
		return fmt.Errorf("error parsing list file name template [%s]", c.ListFileNameTemplate)
	}
	if c.Worker <= 0 {
		return errors.New("worker must be > 0")
	}

	abs, err := filepath.Abs(c.Target)
	if err != nil {
		return err
	}
	c.Target = abs

	if c.ArchiveTarget != "" {
		abs, err := filepath.Abs(c.ArchiveTarget)
		if err != nil {
			return err
		}
		c.ArchiveTarget = abs
	}

	if c.Quiet {
		c.Summary = false
		c.Progress = ProgressNone
	}

	if c.Progress != ProgressSimple && c.Progress != ProgressNone {
		c.Progress = ProgressBar
	}
	return nil
}

// Logger get the logger.
func (c *Config) Logger() log.YALI {
	if c.log == nil {
		c.log = log.New(c.Quiet, c.Progress == ProgressSimple)
	}
	return c.log
}

// OutputFormat get the current output format.
func (c *Config) OutputFormat() string {
	if c.PrintFlags != nil && c.PrintFlags.OutputFormat != nil {
		return *c.PrintFlags.OutputFormat
	}
	return ""
}

// RestConfig get the current rest config.
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
