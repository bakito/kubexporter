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

export interface OwnerReference {
  apiVersion: string;
  kind: string;
  name: string;
  uid?: string;
  controller?: boolean;
}

export interface OpenReport {
  apiVersion: string;
  kind: 'Report';
  metadata: {
    name: string;
    namespace?: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
    resourceVersion?: string;
    creationTimestamp?: string;
    ownerReferences?: OwnerReference[];
  };
  source: string;
  summary: ReportSummary;
  results: ReportResult[];
}

// Trend direction for comparing current vs previous values
export type TrendDirection = 'up' | 'down' | 'same' | 'none';

// Parsed kubexporter export statistics from a report
export interface ExportStats {
  kinds: number;
  resources: number;
  size: number;
  namespaces: number;
  errors: number;
  duration: string;
  prevKinds?: number;
  prevResources?: number;
  prevSize?: number;
  prevNamespaces?: number;
  prevErrors?: number;
}

// Per-resource-kind details
export interface ResourceKindStats {
  groupKind: string;
  instances: number;
  exportedInstances: number;
  exportedSizeBytes: number;
  queryDuration: string;
  exportDuration: string;
  error?: string;
  prevInstances?: number;
  prevExportedInstances?: number;
  prevExportedSizeBytes?: number;
}
