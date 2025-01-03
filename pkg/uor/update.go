package uor

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/bakito/kubexporter/pkg/client"
	"github.com/bakito/kubexporter/pkg/render"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/pkg/utils"
	"github.com/olekukonko/tablewriter"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	if err != nil {
		return err
	}

	ac, err := client.NewApiClient(config)
	if err != nil {
		return err
	}

	table := render.Table()
	table.SetHeader([]string{"File", "Owner Kind", "Owner Name", "UID From", "UID To"})

	ctx := context.TODO()
	for _, file := range files {
		err2 := updateFile(ctx, config, file, ac, table)
		if err2 != nil {
			return err2
		}
	}

	if table.NumLines() == 0 {
		println("No changed owner references found")
	} else {
		table.Render()
	}
	return nil
}

func updateFile(
	ctx context.Context,
	config *types.Config,
	file string,
	ac *client.ApiClient,
	table *tablewriter.Table,
) error {
	fileName := strings.Replace(file, config.Target+"/", "", 1)
	us, err := utils.ReadFile(file)
	if err != nil {
		return err
	}
	refs := us.GetOwnerReferences()
	owners := make(map[string]*unstructured.Unstructured)
	changed := false
	if len(refs) > 0 {
		for _, ref := range refs {
			owner, err := findOwner(ctx, ac, owners, &ref, us)
			if err != nil {
				errMsg := "<ERROR>"
				if errors.IsNotFound(err) {
					errMsg = "<NOT FOUND>"
				}
				table.Append([]string{
					fileName,
					ref.Kind,
					ref.Name,
					string(ref.UID),
					errMsg,
				})
				continue
			}

			if ref.UID != owner.GetUID() {
				table.Append([]string{
					fileName,
					ref.Kind,
					ref.Name,
					string(ref.UID),
					string(owner.GetUID()),
				})
				ref.UID = owner.GetUID()
				changed = true
			}
		}
		if changed {
			us.SetOwnerReferences(refs)
			err := utils.WriteFile(config.PrintFlags, file, us)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func findOwner(
	ctx context.Context,
	ac *client.ApiClient,
	owners map[string]*unstructured.Unstructured,
	ref *v1.OwnerReference,
	us *unstructured.Unstructured,
) (*unstructured.Unstructured, error) {
	key := us.GetNamespace() + "#" + ref.APIVersion + "#" + ref.Name
	if owner, ok := owners[key]; ok {
		return owner, nil
	}

	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, err
	}
	mapping, err := ac.Mapper.RESTMapping(schema.GroupKind{
		Group: gv.Group,
		Kind:  ref.Kind,
	}, gv.Version)
	if err != nil {
		return nil, err
	}
	owner, err := ac.Client.Resource(mapping.Resource).Namespace(us.GetNamespace()).Get(ctx, ref.Name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	owners[key] = owner
	return owner, nil
}
