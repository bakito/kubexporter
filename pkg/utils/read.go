package utils

import (
	"bufio"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
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

func WriteFile(printFlags *genericclioptions.PrintFlags, file string, us *unstructured.Unstructured) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()
	err = PrintObj(printFlags, us, f)
	if err != nil {
		return err
	}
	return nil
}

// PrintObj print the given object
func PrintObj(printFlags *genericclioptions.PrintFlags, ro runtime.Object, out io.Writer) error {
	p, err := printFlags.ToPrinter()
	if err != nil {
		return err
	}
	return p.PrintObj(ro, out)
}
