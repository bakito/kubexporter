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