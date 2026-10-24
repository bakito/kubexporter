// Types matching the OpenReports v1alpha1 API
// https://github.com/openreports/reports-api

export type ResultType = 'pass' | 'fail' | 'warn' | 'error' | 'skip';
export type ResultSeverity = 'info' | 'low' | 'medium' | 'high' | 'critical';

export interface ReportResult {
  source: string;
  policy: string;
  rule: string;
  category: string;
  severity: ResultSeverity;
  description: string;
  properties: Record<string, string>;
  result: ResultType;
  timestamp: string;
}

export interface ReportSummary {
  pass: number;
  fail: number;
  warn: number;
  error: number;
  skip: number;
}

export interface OpenReport {
  apiVersion: string;
  kind: 'Report' | 'ClusterReport';
  metadata: {
    name: string;
    namespace?: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
    resourceVersion?: string;
    creationTimestamp?: string;
  };
  source: string;
  summary: ReportSummary;
  results: ReportResult[];
}

// Parsed kubexporter export statistics from a report
export interface ExportStats {
  kinds: number;
  pages: number;
  resources: number;
  size: number;
  namespaces: number;
  errors: number;
  duration: string;
}

// Per-resource-kind details
export interface ResourceKindStats {
  groupKind: string;
  instances: number;
  exportedInstances: number;
  pages: number;
  exportedSizeBytes: number;
  queryDuration: string;
  exportDuration: string;
  error?: string;
}
