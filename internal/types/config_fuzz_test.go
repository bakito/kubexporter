package types

import (
	"testing"

	"github.com/ghodss/yaml"
)

func FuzzConfigYAML(f *testing.F) {
	f.Add([]byte("target: /tmp\n"))
	f.Add([]byte("namespace: default\n"))
	f.Add([]byte(""))
	f.Add([]byte("invalid: yaml: ["))
	f.Add([]byte("worker: -1\narchive: true\n"))

	f.Fuzz(func(_ *testing.T, data []byte) {
		config := &Config{}
		_ = yaml.Unmarshal(data, config)
	})
}
