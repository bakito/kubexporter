import { OpenReport, ExportStats, ResourceKindStats, OwnerReference, TrendDirection } from './types';

// Extract the summary-level export stats from a kubexporter report
export function parseExportStats(report: OpenReport): ExportStats | null {
  const stats: ExportStats = {
    kinds: 0,
    pages: 0,
    resources: 0,
    size: 0,
    namespaces: 0,
    errors: 0,
    duration: '0s',
  };

  for (const result of report.results) {
    // Only process summary-level results (policy: kubexporter/summary)
    if (result.policy !== 'kubexporter/summary') {
      continue;
    }

    const value = result.properties?.value;
    const prevValue = result.properties?.prevValue;
    if (value === undefined) {
      continue;
    }

    switch (result.rule) {
      case 'kinds':
        stats.kinds = parseInt(value, 10) || 0;
        if (prevValue) stats.prevKinds = parseInt(prevValue, 10) || 0;
        break;
      case 'pages':
        stats.pages = parseInt(value, 10) || 0;
        if (prevValue) stats.prevPages = parseInt(prevValue, 10) || 0;
        break;
      case 'resources':
        stats.resources = parseInt(value, 10) || 0;
        if (prevValue) stats.prevResources = parseInt(prevValue, 10) || 0;
        break;
      case 'size':
        stats.size = parseInt(value, 10) || 0;
        if (prevValue) stats.prevSize = parseInt(prevValue, 10) || 0;
        break;
      case 'namespaces':
        stats.namespaces = parseInt(value, 10) || 0;
        if (prevValue) stats.prevNamespaces = parseInt(prevValue, 10) || 0;
        break;
      case 'errors':
        stats.errors = parseInt(value, 10) || 0;
        if (prevValue) stats.prevErrors = parseInt(prevValue, 10) || 0;
        break;
      case 'duration':
        stats.duration = value;
        break;
    }
  }

  return stats;
}

// Extract per-resource-kind stats from a kubexporter report
export function parseResourceKindStats(report: OpenReport): ResourceKindStats[] {
  const list: ResourceKindStats[] = [];

  for (const result of report.results) {
    // Only process per-resource results (policy: kubexporter/resource)
    if (result.policy !== 'kubexporter/resource') {
      continue;
    }

    const props = result.properties || {};
    const stats: ResourceKindStats = {
      groupKind: result.rule,
      instances: parseInt(props.instances, 10) || 0,
      exportedInstances: parseInt(props.exportedInstances, 10) || 0,
      pages: parseInt(props.pages, 10) || 0,
      exportedSizeBytes: parseInt(props.exportedSizeBytes, 10) || 0,
      queryDuration: props.queryDuration || '0s',
      exportDuration: props.exportDuration || '0s',
      error: props.error || undefined,
      prevInstances: props.prevInstances ? parseInt(props.prevInstances, 10) : undefined,
      prevExportedInstances: props.prevExportedInstances ? parseInt(props.prevExportedInstances, 10) : undefined,
      prevExportedSizeBytes: props.prevExportedSizeBytes ? parseInt(props.prevExportedSizeBytes, 10) : undefined,
    };

    list.push(stats);
  }

  return list;
}

// Format bytes to human-readable string
export function formatBytes(bytes: number): string {
  if (bytes === 0) {
    return '0 B';
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));

  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

// Format a number with thousands separator
export function formatNumber(n: number): string {
  return n.toLocaleString('en-US');
}

// Get the last export timestamp from report annotations
export function getLastExportTime(report: OpenReport): string | null {
  return report.metadata.annotations?.['kubexporter.bakito.net/last-export'] || null;
}

// Check if a report is from kubexporter
export function isKubexporterReport(report: OpenReport): boolean {
  return report.source === 'kubexporter';
}

// Get the CronJob owner reference from a report, if present
export function getCronJobOwner(report: OpenReport): OwnerReference | null {
  const refs = report.metadata.ownerReferences;
  if (!refs) {
    return null;
  }
  const cronJob = refs.find((ref) => ref.kind === 'CronJob');
  return cronJob || null;
}

// Calculate the trend direction between current and previous values
export function getTrend(current: number, previous?: number): TrendDirection {
  if (previous === undefined) {
    return 'none';
  }
  if (current > previous) {
    return 'up';
  }
  if (current < previous) {
    return 'down';
  }
  return 'same';
}
