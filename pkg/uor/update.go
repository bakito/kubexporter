package uor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bakito/kubexporter/pkg/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func Update(config *types.Config) error {
	err := config.Validate()
	if err != nil {
		return err
	}

	var files []string
	err = filepath.Walk(config.Target, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == "."+config.OutputFormat() {
			files = append(files, path)
		}

		return nil
	})
	rc, err := config.RestConfig()
	if err != nil {
		return err
	}

	client, err := dynamic.NewForConfig(rc)
	if err != nil {
		return err
	}

	dcl, err := discovery.NewDiscoveryClientForConfig(rc)
	if err != nil {
		return err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dcl))
	ctx := context.TODO()
	for _, file := range files {
		us, err := read(file)
		if err != nil {
			return err
		}
		refs := us.GetOwnerReferences()
		if len(refs) > 0 {
			for i := range refs {
				ref := &refs[i]
				group, version := groupVersion(ref)
				mapping, err := mapper.RESTMapping(schema.GroupKind{
					Group: group,
					Kind:  ref.Kind,
				}, version)
				if err != nil {
					return err
				}
				owner, err := client.Resource(mapping.Resource).Namespace(us.GetNamespace()).Get(ctx, ref.Name, v1.GetOptions{})

				if err != nil {
					return err
				}
				fmt.Printf("Changed owner of %s.%s %s/%s\n\t%s.%s %s/%s from %s -> %s\n",
					us.GetAPIVersion(),
					us.GetKind(),
					us.GetNamespace(),
					us.GetName(),
					owner.GetAPIVersion(),
					owner.GetKind(),
					owner.GetNamespace(),
					owner.GetName(),
					ref.UID, owner.GetUID())
				ref.UID = owner.GetUID()
			}
			us.SetOwnerReferences(refs)
		}
	}
	return nil

}

func groupVersion(reference *v1.OwnerReference) (string, string) {
	gv := strings.Split(reference.APIVersion, "/")
	var group string
	var version string
	if len(gv) > 1 {
		group = gv[0]
		version = gv[1]
	} else {
		version = gv[0]
	}
	return group, version
}

func read(file string) (*unstructured.Unstructured, error) {
	us := &unstructured.Unstructured{}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bufio.NewReader(f), 20)
	err = decoder.Decode(us)
	if err != nil {
		return nil, err
	}
	return us, nil
}
