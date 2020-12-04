[![Go](https://github.com/bakito/kubexporter/workflows/Go/badge.svg)](https://github.com/bakito/kubexporter/actions?query=workflow%3AGo)
[![Docker Repository on Quay](https://quay.io/repository/bakito/kubexporter/status "Docker Repository on Quay")](https://quay.io/repository/bakito/kubexporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/bakito/kubexporter)](https://goreportcard.com/report/github.com/bakito/kubexporter)
[![GitHub Release](https://img.shields.io/github/release/bakito/kubexporter.svg?style=flat)](https://github.com/bakito/kubexporter/releases)

# KubExporter

KubExporter allows you to export resources from kubernetes as yaml/json files.

The configuration allows customization on which resources and which fields to exclude.

## Usage

```bash
kubexporter --config config.yaml
```
![kubexporter](doc/kubexporter.gif)


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
    - Secret
    - apps.ReplicaSet
    - authentication.k8s.io.TokenReview
    - authorization.k8s.io.LocalResourceAccessReview
    - authorization.k8s.io.LocalSubjectAccessReview
    - authorization.k8s.io.ResourceAccessReview
    - authorization.k8s.io.SelfSubjectAccessReview
    - authorization.k8s.io.SelfSubjectRulesReview
    - authorization.k8s.io.SubjectAccessReview
    - authorization.openshift.io.LocalResourceAccessReview
    - authorization.openshift.io.LocalSubjectAccessReview
    - authorization.openshift.io.ResourceAccessReview
    - authorization.openshift.io.SelfSubjectRulesReview
    - authorization.openshift.io.SubjectAccessReview
    - authorization.openshift.io.SubjectRulesReview
    - batch.Job
    - build.openshift.io.Build
    - events.k8s.io.Event
    - extensions.ReplicaSet
    - image.openshift.io.Image
    - image.openshift.io.ImageSignature
    - image.openshift.io.ImageStreamImage
    - image.openshift.io.ImageStreamImage
    - image.openshift.io.ImageStreamImport
    - image.openshift.io.ImageStreamMapping
    - image.openshift.io.ImageStreamTag
    - security.openshift.io.PodSecurityPolicyReview
    - security.openshift.io.PodSecurityPolicySelfSubjectReview
    - security.openshift.io.PodSecurityPolicySubjectReview
    - user.openshift.io.UserIdentityMapping
  fields: # list fields that should be removed for all resources before exported
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
    image.openshift.io.ImageStream:
      - [ annotations, "openshift.io/image.dockerRepositoryCheck" ]

```
