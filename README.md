[![Go](https://github.com/bakito/kubexporter/workflows/Go/badge.svg)](https://github.com/bakito/kubexporter/actions?query=workflow%3AGo)
[![GitHub Release](https://img.shields.io/github/release/bakito/kubexporter.svg?style=flat)](https://github.com/bakito/kubexporter/releases)
[![Coverage Status](https://coveralls.io/repos/github/bakito/kubexporter/badge.svg?branch=main)](https://coveralls.io/github/bakito/kubexporter?branch=main)
[![Static Badge](https://img.shields.io/badge/try_me-on_Killercoda-black)](https://killercoda.com/bakito/scenario/kubernetes-kubexporter)


<div align="right">
  <img src="docs/icons/kubexporter.png" alt="kubexporter" width="100"/>
</div>

# KubExporter

KubExporter allows you to export resources from kubernetes as yaml/json files.

The configuration allows customization on which resources and which fields to exclude.

## Install

Download the latest binary from https://github.com/bakito/kubexporter/releases.

[![Packaging status](https://repology.org/badge/vertical-allrepos/kubexporter.svg)](https://repology.org/project/kubexporter/versions)

### Brew

```bash
# Add the tap
brew tap bakito/tap
brew trust bakito/tap

# install kubexporter 
brew install --cask kubexporter
```

### Use as kubectl plugin

Rename the binary to kubectl-exporter.

```bash
kubectl exporter ...
```

## Usage

<!-- cli-doc-start -->
```
easily export kubernetes resources

Usage:
  kubexporter [flags]
  kubexporter [command]

Available Commands:
  completion              Generate the autocompletion script for the specified shell
  decrypt                 Decrypt secrets in exported resource files
  encrypt                 Encrypt secrets in exported resource files
  help                    Help about any command
  update-owner-references Update owner references of an export against the current cluster

Flags:
  -a, --archive                        Create a tar.gz archive
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --as-user-extra stringArray      User extras to impersonate for the operation, this flag can be repeated to specify multiple values for the same key.
      --certificate-authority string   Path to a cert file for the certificate authority
  -c, --clear-target                   Clear the target directory before exporting
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --config string                  config file
      --context string                 The name of the kubeconfig context to use
      --created-within duration        The max allowed age duration for the resources
      --disable-compression            If true, opt-out of response compression for all requests to the server
  -d, --exclude-defaults               If enabled, default excludes will be applied. [apps.ControllerRevision, apps.ReplicaSet, batch.Job, Pod, ReplicationController, discovery.k8s.io.EndpointSlice, Endpoints, Event, events.k8s.io.Event, coordination.k8s.io.Lease, metrics.k8s.io.NodeMetrics, metrics.k8s.io.PodMetrics, ComponentStatus, Secret, LocalSubjectAccessReview, SelfSubjectAccessReview, SelfSubjectRulesReview, SubjectAccessReview, TokenReview, Binding]
  -e, --exclude-kinds strings          List all kinds to be excluded
  -h, --help                           help for kubexporter
      --include-cluster-resources      Export cluster-scoped resources too, when a namespace filter is active
  -i, --include-kinds strings          List all kinds to be included
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -l, --lists                          Export as lists instead of individual files
  -n, --namespace strings              A single namespace (default all)
      --otlp-metrics                   OTLP Metrics are enabled
  -o, --output string                  Output format. One of: (json, yaml, kyaml). (default "yaml")
  -p, --progress string                Progress mode bar|bubbles|simple|none (default "bar")
  -q, --quiet                          Output is prevented
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --show-managed-fields            If true, keep the managedFields when printing objects in JSON or YAML format.
      --size                           Print the size of the exported files
      --summary                        If enabled, a summary is printed
  -t, --target string                  The target directory (default "exports")
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
  -v, --verbose                        Errors during export are listed in summary
      --version                        version for kubexporter
  -w, --worker int                     The number of parallel worker (default 1)

Use "kubexporter [command] --help" for more information about a command.
```
<!-- cli-doc-end -->

[![asciicast](https://asciinema.org/a/J793zgHiRBgDTgWbKjHrsM8YL.svg)](https://asciinema.org/a/J793zgHiRBgDTgWbKjHrsM8YL)

## Configuration

### Config

KubExporter exports by default all resources and allows excluding unwanted resources.
The benefit is that new custom resource definitions are automatically considered in the export.

Example configuration: [config.yaml](config.yaml)

Config Documentation:

<!-- yaml-doc-start -->
```yaml
# Excluded resources (struct)
excluded:
  # List all kinds to be excluded ([]string)
  kinds:
  # List fields that should be removed for all resources before exported; slices are also traversed ([][]string)
  fields:
  # Kind specific excluded fields (map[string:[][]string])
  kindFields:
  # Allows to exclude single instances with certain field values (map[string:[]struct])
  kindByField:
  # List of fields to be preserved (struct)
  preservedFields:
    # Fields to be preserved ([][]string)
    fields:
# Included resources (struct)
included:
  # List all kinds to be included ([]string)
  kinds:
# The max allowed age duration for the resources (int64)
createdWithin:
# Consider owner references for not excluded resources (bool)
considerOwnerReferences:
# Field masking config (struct)
masked:
  # The replacement value for masked fields (string)
  replacement:
  # The checksum algorithm to use for masked fields md5|sha1|sha256 (default md5) (string)
  checksum:
  # The fields to mask for each kind (map[string:[][]string])
  kindFields:
# Field encryption config (struct)
encrypted:
  # The AES key to use for field encryption (string)
  aesKey:
  # The fields to encrypt for each kind (map[string:[][]string])
  kindFields:
# sort the slice field value before exporting (map[string:[][]string])
sortSlices:
# Custom resource file name template (string)
fileNameTemplate:
# Custom resource list file name template (string)
listFileNameTemplate:
# Export as lists instead of individual files (bool)
asLists:
# Kubernetes query page size (0 use default) (int)
queryPageSize:
# The target directory (string)
target:
# Clear the target directory before exporting (bool)
clearTarget:
# If enabled, a summary is printed (bool)
summary:
# Progress mode bar|bubbles|simple|none (string)
progress:
# A single namespace (default all) (string)
namespace:
# Multiple namespaces (joined with namespace, if both are set) ([]string)
namespaces:
# Export cluster-scoped resources too, when a namespace filter is active (bool)
includeClusterResources:
# The number of parallel worker (int)
worker:
# Create a tar.gz archive (bool)
archive:
# Number of days to keep old archives (int)
archiveRetentionDays:
# The target directory for the archive(default "exports") (string)
archiveTarget:
# S3 Configuration to upload the archive to an S3 compatible storage (struct)
s3:
  # S3 Endpoint (string)
  endpoint:
  # Access key ID (string)
  accessKeyID:
  # Secret access key (string)
  secretAccessKey:
  # Session token (optional) (string)
  token:
  # Use HTTPS (default true) (bool)
  secure:
  # Bucket name (string)
  bucket:
# Google storage bucket configuration (struct)
gcs:
  # Bucket name (string)
  bucket:
# Metrics configuration (struct)
metrics:
  # OTel Metrics configuration (struct)
  otlp:
    # OTLP Metrics are enabled (bool)
    enabled:
    # OTLP Metrics endpoint (string)
    endpoint:
    # OTLP Metrics insecure (bool)
    insecure:
# Output is prevented (bool)
quiet:
# Errors during export are listed in summary (bool)
verbose:
# Print the size of the exported files (bool)
printSize:
```
<!-- yaml-doc-end -->

### S3

You can configure `kubexporter` to upload the created archive to an S3 compatible storage.

The following fields are available for S3 configuration:

* `endpoint`: The S3 endpoint.
* `accessKeyID`: The access key ID.
* `secretAccessKey`: The secret access key.
* `token`: The session token (optional).
* `secure`: Set to `true` for HTTPS, `false` for HTTP.
* `bucket`: The name of the S3 bucket.

#### Authentication

Credentials must be provided in the config file. Environment variables are not automatically used if these fields are
set (even if empty).

Example:

```yaml
s3:
  endpoint: <your-s3-endpoint>
  accessKeyID: <your-access-key-id>
  secretAccessKey: <your-secret-access-key>
  token: <your-session-token> # Optional
  secure: true # Use HTTPS (default)
  bucket: <your-bucket-name>
```

### GCS

You can configure `kubexporter` to upload the created archive to a GCS bucket.

The following fields are available for GCS configuration:

* `bucket`: The name of the GCS bucket.

#### Authentication

Authentication to Google Cloud Storage is handled automatically
via [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/application-default-credentials).

You can configure ADC in one of the following ways:

* **Service Account Key File:** Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the path of the JSON
  file that contains your service account key.
  ```bash
  export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/keyfile.json"
  ```
* **gcloud CLI:** Authenticate with the gcloud CLI.
  ```bash
  gcloud auth application-default login
  ```
* **Workload Identity (Recommended for GKE):** When running in a GKE cluster, the recommended way to authenticate is by
  using Workload Identity. This allows your Kubernetes pod to impersonate a Google Service Account without needing to
  handle service account keys.

Example:

```yaml
gcs:
  bucket: <your-bucket-name>
```


### Metrics

kubexporter supports pushing metrics to OTel collectors.

```yaml
metrics:
  otlp:
    enabled: true
    endpoint: localhost:4317
    insecure: true
```

#### Metrics 

<!-- metrics-doc-start -->
| Metric | Description |
| ------ | ----------- |
| kubexporter.duration_seconds | Total export duration in seconds |
| kubexporter.errors | Number of errors encountered during export |
| kubexporter.exported_resources | Total number of exported resources |
| kubexporter.exported_size_bytes | Total size of exported resources in bytes |
| kubexporter.kinds | Number of kinds processed |
| kubexporter.namespaces | Number of namespaces containing exported resources |
| kubexporter.query_pages | Total number of query pages requested |
| kubexporter.resource.export_duration_seconds | Export duration per kind in seconds |
| kubexporter.resource.exported_instances | Number of exported resource instances per kind |
| kubexporter.resource.exported_size_bytes | Size of exported resources per kind in bytes |
| kubexporter.resource.instances | Number of resource instances found per kind |
| kubexporter.resource.query_duration_seconds | Query duration per kind in seconds |
| kubexporter.resource.query_pages | Number of query pages per kind |
<!-- metrics-doc-end -->

#### Grafana Dashboard

A pre-configured Grafana sample dashboard is available at [examples/grafana/dashboard.json](examples/grafana/dashboard.json).

#### Authentication

OTLP Authentication headers can be defined via env variables with prefix `KUBEXPORTER_METRICS_OTLP_HEADER_`

E.g: `KUBEXPORTER_METRICS_OTLP_HEADER_Authorization="Bearer <yourToken>`

### Update Owner References

Allows updating Owner references against a running cluster.

```shell
kubexporter update-owner-references

 FILE                                                                                 OWNER KIND  OWNER NAME                                 UID FROM                              UID TO                               
 cert-manager/cilium.io.CiliumEndpoint.cert-manager-cainjector-7fd8f6bbbf-9nlf2.yaml  Pod         cert-manager-cainjector-7fd8f6bbbf-9nlf2   1d494969-hhhh-4c79-96d4-25d31c66c895  1d494969-db54-4c79-96d4-25d31c66c895 
 cert-manager/cilium.io.CiliumEndpoint.cert-manager-webhook-787cd749dc-7sfvq.yaml     Pod         cert-manager-webhook-787cd749dc-7sfvq-XXX  eeeb48d9-751c-4aa9-9389-6aab845dba1e  <NOT FOUND>      
```

### Decrypt encrypted values

Exported files with encrypted values can be decrypted with the decrypt command.

The aes key can b provided via arg `--aes-key`, env variable `KUBEXPORTER_AES_KEY`. If not provided the key can be
entered via password prompt.

1 - n file paths are defined via command arguments.

```shell
kubexporter decrypt exports/argocd/Secret.argocd-secret.yaml

 FILE                                      NAMESPACE  KIND    NAME           DECRYPTED FIELDS
 exports/argocd/Secret.argocd-secret.yaml  argocd     Secret  argocd-secret                 5

```

#### Decrypt multiple files

```shell
kubexporter decrypt $(ls exports/argocd/Secret*)
```
