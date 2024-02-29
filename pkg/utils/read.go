package utils

import (
	"bufio"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func ReadFile(file string) (*unstructured.Unstructured, error) {
	us := &unstructured.Unstructured{}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewYAMLOrJSONDecoder(bufio.NewReader(f), 20)
	err = decoder.Decode(us)
	if err != nil {
		return nil, err
	}
	return us, nil
}
