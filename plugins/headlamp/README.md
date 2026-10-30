# KubExporter Headlamp Plugin

A [Headlamp](https://headlamp.dev/) plugin that visualizes kubexporter export statistics from OpenReports CRDs.

## Overview

This plugin adds a "KubExporter" dashboard to Headlamp that displays:

- **Export Statistics Cards**: Resources exported, kinds processed, namespaces, export size, duration, and errors
- **Resource Kinds Table**: Per-resource-kind breakdown with instances, size, and query/export time

## Prerequisites

1. **Headlamp** installed and running (desktop app or server)
2. **OpenReports CRD** installed in your cluster
3. **kubexporter** has been run with the `--openreports` flag

## Installation

### Option 1: Development Mode

```bash
cd plugins/headlamp
npm install
npm run start
```

Then point Headlamp to the plugin directory:

```bash
headlamp --plugins=plugins/headlamp
```

Or in Headlamp settings, add the plugin path.

### Option 2: Build and Install

```bash
cd plugins/headlamp
npm install
npm run build
```

Then load the built plugin in Headlamp:

```bash
headlamp --plugins=plugins/headlamp/dist
```

### Option 3: Artifact Hub

The plugin is published to [Artifact Hub](https://artifacthub.io/) and can be installed directly from the Headlamp Plugin Catalog.

## Releasing

To publish a new version, run this from the kubexporter repository root:

```bash
make release-headlamp-plugin
```

Run it on the `main` branch. When prompted, accept the proposed next version (derived from the latest `headlamp-plugin-v*` tag) or enter another one (for example, `v0.0.2`). The target creates and pushes a signed tag named `headlamp-plugin-v0.0.2`. That tag starts the GitHub Actions release workflow, which updates the plugin version, builds the archive, calculates its checksum, and updates this package's `artifacthub-pkg.yml`.

The workflow attaches `kubexporter-headlamp-plugin-0.0.2.tar.gz` to the GitHub release, then publishes `artifacthub-pkg.yml` and this README to `bakito/headlamp-plugins` under `kubexporter/0.0.2/`. The repository must have a `HEADLAMP_PLUGINS_TOKEN` secret with permission to push to `bakito/headlamp-plugins`.

The version in `package.json` is updated in the workflow; no manual version edit or archive upload is needed. The normal kubexporter `vX.Y.Z` release process is unchanged.

For more details, see the [Headlamp Publishing Plugins guide](https://headlamp.dev/docs/latest/development/plugins/publishing/).

## Creating Reports

Run kubexporter with the OpenReports flag to create a report:

```bash
kubexporter --openreports --openreports-namespace=default
```

The plugin will automatically detect and display the most recent kubexporter report.

## Dashboard Features

### Dashboard Links

If the kubexporter report includes the `kubexporter.bakito.ch/headlamp-link` annotation, each configured link appears as a button in the dashboard header and opens in a new browser tab. The annotation accepts a JSON array of links with a title, an HTTP(S) URL, and an optional icon. Icons may be Iconify IDs or Font Awesome classes. A legacy single HTTP(S) URL remains supported and is shown as an "Open in Headlamp" button.

Set the annotation via the `annotations` config field:

```yaml
openReports:
  enabled: true
  annotations:
    kubexporter.bakito.ch/headlamp-link: '[{"title":"Dashboard","url":"https://headlamp.local/plugins/kubexporter","icon":"fa-solid fa-chart-line"}]'
```

### Statistics Cards

Six cards showing the key export metrics:
- **Resources**: Total resources exported
- **Kinds**: Number of resource kinds processed
- **Namespaces**: Number of namespaces covered
- **Size**: Total exported data size
- **Duration**: Export run duration
- **Errors**: Number of errors (red if > 0)


### Resource Kinds Table

Sortable, paginated table showing per-resource-kind details:
- Kind name (group.kind format)
- Total instances found
- Exported instances
- Exported size
- Query duration
- Export duration

## Architecture

The plugin fetches OpenReports `Report` CRDs directly from the Kubernetes API via Headlamp's proxy. It filters for reports with `source: kubexporter` and parses the structured results into dashboard components.

## Project Structure

```
plugins/headlamp/
├── package.json              # Plugin metadata and dependencies
├── tsconfig.json             # TypeScript configuration
├── README.md                 # This file
└── src/
    ├── index.tsx             # Plugin entry point (sidebar + route registration)
    ├── KubExporterDashboard.tsx  # Main dashboard page component
    ├── types.ts              # TypeScript types for OpenReports data
    ├── utils.ts              # Parsing and formatting utilities
    └── components/
        ├── StatsCards.tsx    # Statistics cards grid
        └── ResourceTable.tsx # Per-resource-kind details table
```

## License

Apache-2.0
