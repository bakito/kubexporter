[![Go](https://github.com/bakito/kubexporter/workflows/Go/badge.svg)](https://github.com/bakito/kubexporter/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/bakito/kubexporter)](https://goreportcard.com/report/github.com/bakito/kubexporter)
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

# install kubexporter 
brew install --cask kubexporter
```

### Snap

```bash
sudo snap install kubexporter
```

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
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --cache-dir string               Default cache directory (default "/home/bakito/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
  -c, --clear-target                   If enabled, the target dir is deleted before running the new export
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --config string                  config file
      --context string                 The name of the kubeconfig context to use
      --created-within duration        The max allowed age duration for the resources
      --disable-compression            If true, opt-out of response compression for all requests to the server
  -e, --exclude-kinds strings          Do not export excluded kinds
  -h, --help                           help for kubexporter
  -i, --include-kinds strings          Export only included kinds, if included kinds are defined, excluded will be ignored
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -l, --lists                          If enabled, all resources are exported as lists instead of individual files
  -n, --namespace string               If present, the namespace scope for this CLI request
  -o, --output string                  Output format. One of: (json, yaml). (default "yaml")
  -p, --progress string                Progress mode bar|simple|none (default bar)  (default "bar")
  -q, --quiet                          If enabled, output is prevented
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --show-managed-fields            If true, keep the managedFields when printing objects in JSON or YAML format.
      --summary                        If enabled, a summary is printed
  -t, --target string                  Set the target directory (default exports) (default "exports")
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
  -v, --verbose                        If enabled, errors during export are listed in summary
      --version                        version for kubexporter
  -w, --worker int                     The number of worker to use for the export (default 1)

```

[![asciicast](https://asciinema.org/a/J793zgHiRBgDTgWbKjHrsM8YL.svg)](https://asciinema.org/a/J793zgHiRBgDTgWbKjHrsM8YL)

## Configuration

### Config

KubExporter exports by default all resources and allows to exclude unwanted resources.
The benefit is that new custom resource definitions are automatically considered in the export.

Example configuration

```yaml
# print a summary
summary: true
# print progress (bar|simple|none)
progress: bar
# create an archive
archive: true
# S3 Configuration to upload the archive to an S3 compatible storage
#s3:
#  endpoint: <your-s3-endpoint>
#  accessKeyID: <your-access-key-id>
#  secretAccessKey: <your-secret-access-key>
#  token: <your-session-token> # Optional
#  secure: true # Use HTTPS (default)
#  bucket: <your-bucket-name>
#gcs:
#  bucket: <your-bucket-name>

# define a single namespace (default all)
namespace:
# define the number of parallel worker
worker: 1
# export as lists
asLists: false
# enable pagination on queries (only supported when asLists = false)
#queryPageSize: 1000
# clear the target directory before exporting
clearTarget: true
excluded:
  # list all kinds to be excluded
  kinds:
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
  # list fields that should be removed for all resources before exported; slices are also traversed
  fields:
    - [status]
    - [metadata, uid]
    - [metadata, selfLink]
    - [metadata, resourceVersion]
    - [metadata, creationTimestamp]
    - [metadata, generation]
    - [metadata, annotations, "kubectl.kubernetes.io/last-applied-configuration"]
  # kind specific excluded fields
  kindFields:
    Service:
      - [spec, clusterIP]
  # allows to exclude single instances with certain field values
  kindByField:
    Service:
      - field: [metadata, name]
        # the value is compared to the string representation of the actual kind value
        values: [exclude-me-1, exclude-me-2]
    Secret:
      - field: [type]
        # exclude helm secrets
        values: ['helm.sh/release', 'helm.sh/release.v1']
# excludes resources if the owner reference kind is excluded
considerOwnerReferences: false
# mask certain fields 
masked:
  # the replacement string to be used for masked fields (default '***')
  replacement: '***'
  # generate a checksum from the value to be masked value instead of the replacement. (supported 'md5', 'sha1', 'sha256')  
  checksum: ''
  # kind specific fields that should be masked
  kindFields:
    Secret:
      - [data]
# encrypt certain fields 
#encrypted:
#  # the aes key to use to encrypt the field values. The key can also be provided via env variable 'KUBEXPORTER_AES_KEY'
#  aesKey: '***'
#  # kind specific fields that should be encrypted. NOTE: if the same fields or a parent branch is also masked, masking wins over encryption.
#  kindFields:
#    Secret:
#      - [ data ]

# sort the slice field value before exporting
sortSlices:
  User:
    - [roles]
```

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
