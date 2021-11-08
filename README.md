[![Go](https://github.com/bakito/kubexporter/workflows/Go/badge.svg)](https://github.com/bakito/kubexporter/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/bakito/kubexporter)](https://goreportcard.com/report/github.com/bakito/kubexporter)
[![GitHub Release](https://img.shields.io/github/release/bakito/kubexporter.svg?style=flat)](https://github.com/bakito/kubexporter/releases)
[![Coverage Status](https://coveralls.io/repos/github/bakito/kubexporter/badge.svg?branch=main)](https://coveralls.io/github/bakito/kubexporter?branch=main)
# KubExporter

KubExporter allows you to export resources from kubernetes as yaml/json files.

The configuration allows customization on which resources and which fields to exclude.

## Install

Downlad the latest binary from https://github.com/bakito/kubexporter/releases.

### Use as kubectl plugin

Rename the binary to kubectl-exporter.
```bash
kubectl exporter ...
```


## Usage

```bash
Usage:
  kubexporter [flags]
  
Flags:
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "C:\\develop\\home\\.kube\\cache")
      --certificate-authority string   Path to a cert file for the certificate authority
  -c, --clear-target                   If enabled, the target dir is deleted before running the new export
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --config string                  config file
      --context string                 The name of the kubeconfig context to use
  -e, --exclude-kinds strings          Do not export excluded kinds
  -h, --help                           help for kubexporter
  -i, --include-kinds strings          Export only included kinds, if included kinds are defined, excluded will be ignored
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -l, --lists                          If enabled, all resources are exported as lists instead of individual files
  -n, --namespace string               If present, the namespace scope for this CLI request
  -o, --output string                  Output format. One of: json|yaml. (default "yaml")
  -p, --progress string                Progress mode bar|simple|none (default bar)  (default "bar")
  -q, --quiet                          If enabled, output is prevented
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --summary                        If enabled, a summary is printed
  -t, --target string                  Set the target directory (default exports)
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
  -v, --verbose                        If enabled, errors during export are listed in summary
      --version                        version for kubexporter
  -w, --worker int                     The number of worker to use for the export (default 1)
```

[![asciicast](https://asciinema.org/a/J793zgHiRBgDTgWbKjHrsM8YL.svg)](https://asciinema.org/a/J793zgHiRBgDTgWbKjHrsM8YL)

### Config

KubExporter exports by default all resources and allows to exclude unwanted resources.
The benefit is that new custom resource definitions are automatically considered in the export.



Example configuration

```yaml
summary: true # print a summary
progress: true # print progress
archive: true # create an archive
namespace: # define a single namespace (default all)
worker: 1 # define the number of parallel worker
asLists: false # export as lists
clearTarget: true # clear the target directory before exporting
excluded:
  kinds: # list all kinds to be excluded
    - Binding
    - ComponentStatus
    - Endpoints
    - Event
    - LimitRange
    - LocalSubjectAccessReview
    - PersistentVolume
    - Pod
    - ReplicationController
    - ReplicationControllerDummy
    - RoleBindingRestriction
    - apps.ReplicaSet
    - batch.Job
    - events.k8s.io.Event
    - extensions.ReplicaSet
  fields: # list fields that should be removed for all resources before exported; slices are also traversed
    - [ status ]
    - [ metadata, uid ]
    - [ metadata, selfLink ]
    - [ metadata, resourceVersion ]
    - [ metadata, creationTimestamp ]
    - [ metadata, generation ]
    - [ metadata, annotations, "kubectl.kubernetes.io/last-applied-configuration" ]
  kindFields: # kind specific excluded fields
    Service:
      - [ spec, clusterIP ]
  kindByField: # allows to exclude single instances wir certain field values
    Service:
    - field: [ metadata, name ]
      values: [ exclude-me-1, exclude-me-2 ] # the value is compared to the string representation of the actual kind value
masked: # mask certain fields 
  replacement: '***'
  kindFields: # kind specific excluded fields with a replacement
    Secret:
      - [ data ]
sortSlices: # sort the slice field value before exporting
  User:
    - [ roles ]
```
