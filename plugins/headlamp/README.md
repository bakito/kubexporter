# Kubexporter Headlamp Plugin

A [Headlamp](https://headlamp.dev/) plugin that visualizes kubexporter export statistics from OpenReports CRDs.

## Overview

This plugin adds a "Kubexporter" dashboard to Headlamp that displays:

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

Coming soon.

## Creating Reports

Run kubexporter with the OpenReports flag to create a report:

```bash
kubexporter --openreports --openreports-namespace=default
```

The plugin will automatically detect and display the most recent kubexporter report.

## Dashboard Features

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
    ├── KubexporterDashboard.tsx  # Main dashboard page component
    ├── types.ts              # TypeScript types for OpenReports data
    ├── utils.ts              # Parsing and formatting utilities
    └── components/
        ├── StatsCards.tsx    # Statistics cards grid
        └── ResourceTable.tsx # Per-resource-kind details table
```

## License

Apache-2.0
