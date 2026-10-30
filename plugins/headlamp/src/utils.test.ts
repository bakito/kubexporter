import { describe, expect, it } from 'vitest';
import { OpenReport, ReportResult } from './types';
import {
  formatBytes,
  formatNumber,
  getCronJobOwner,
  getLastExportTime,
  getReportLinks,
  getTrend,
  isKubExporterReport,
  parseExportStats,
  parseResourceKindStats,
  toIconifyIcon,
} from './utils';

function makeReport(overrides: Partial<OpenReport> = {}): OpenReport {
  return {
    apiVersion: 'openreports.io/v1alpha1',
    kind: 'Report',
    metadata: { name: 'test-report' },
    source: 'kubexporter',
    summary: { pass: 0, fail: 0, warn: 0, error: 0, skip: 0 },
    results: [],
    ...overrides,
  };
}

function makeResult(overrides: Partial<ReportResult> = {}): ReportResult {
  return {
    source: 'kubexporter',
    policy: 'kubexporter/summary',
    rule: 'resources',
    category: '',
    severity: 'info',
    description: '',
    properties: {},
    result: 'pass',
    timestamp: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('parseExportStats', () => {
  it('extracts summary values and previous values', () => {
    const stats = parseExportStats(
      makeReport({
        results: [
          makeResult({ rule: 'kinds', properties: { value: '8', prevValue: '7' } }),
          makeResult({ rule: 'resources', properties: { value: '123', prevValue: '100' } }),
          makeResult({ rule: 'size', properties: { value: '2048', prevValue: '1024' } }),
          makeResult({ rule: 'namespaces', properties: { value: '4', prevValue: '3' } }),
          makeResult({ rule: 'errors', properties: { value: '2', prevValue: '1' } }),
          makeResult({ rule: 'duration', properties: { value: '1m2s' } }),
        ],
      }),
    );

    expect(stats).toEqual({
      kinds: 8,
      resources: 123,
      size: 2048,
      namespaces: 4,
      errors: 2,
      duration: '1m2s',
      prevKinds: 7,
      prevResources: 100,
      prevSize: 1024,
      prevNamespaces: 3,
      prevErrors: 1,
    });
  });

  it('ignores non-summary results and keeps defaults for missing values', () => {
    const stats = parseExportStats(
      makeReport({
        results: [
          makeResult({ policy: 'kubexporter/resource', rule: 'resources', properties: { value: '9' } }),
          makeResult({ rule: 'kinds', properties: {} }),
        ],
      }),
    );

    expect(stats).toEqual({ kinds: 0, resources: 0, size: 0, namespaces: 0, errors: 0, duration: '0s' });
  });
});

describe('parseResourceKindStats', () => {
  it('extracts resource results and supplies defaults for optional properties', () => {
    const stats = parseResourceKindStats(
      makeReport({
        results: [
          makeResult({
            policy: 'kubexporter/resource',
            rule: 'apps/Deployment',
            properties: {
              instances: '12',
              exportedInstances: '10',
              exportedSizeBytes: '2048',
              queryDuration: '20ms',
              exportDuration: '40ms',
              error: 'partial export',
              prevInstances: '11',
              prevExportedInstances: '9',
              prevExportedSizeBytes: '1024',
            },
          }),
          makeResult({
            policy: 'kubexporter/resource',
            rule: 'v1/Service',
          }),
          makeResult({ policy: 'kubexporter/summary', rule: 'kinds' }),
        ],
      }),
    );

    expect(stats).toEqual([
      {
        groupKind: 'apps/Deployment',
        instances: 12,
        exportedInstances: 10,
        exportedSizeBytes: 2048,
        queryDuration: '20ms',
        exportDuration: '40ms',
        error: 'partial export',
        prevInstances: 11,
        prevExportedInstances: 9,
        prevExportedSizeBytes: 1024,
      },
      {
        groupKind: 'v1/Service',
        instances: 0,
        exportedInstances: 0,
        exportedSizeBytes: 0,
        queryDuration: '0s',
        exportDuration: '0s',
        error: undefined,
        prevInstances: undefined,
        prevExportedInstances: undefined,
        prevExportedSizeBytes: undefined,
      },
    ]);
  });
});

describe('formatting helpers', () => {
  it.each([
    [0, '0 B'],
    [512, '512 B'],
    [1024, '1.0 KB'],
    [1048576, '1.0 MB'],
  ])('formats %i bytes', (bytes, expected) => {
    expect(formatBytes(bytes)).toBe(expected);
  });

  it('formats numbers with an English thousands separator', () => {
    expect(formatNumber(1234567)).toBe('1,234,567');
  });
});

describe('report metadata helpers', () => {
  it('returns the last export annotation or null when absent', () => {
    expect(
      getLastExportTime(
        makeReport({ metadata: { name: 'test-report', annotations: { 'kubexporter.bakito.net/last-export': '2026-01-01T12:00:00Z' } } }),
      ),
    ).toBe('2026-01-01T12:00:00Z');
    expect(getLastExportTime(makeReport())).toBeNull();
  });

  it('identifies reports by their source', () => {
    expect(isKubExporterReport(makeReport())).toBe(true);
    expect(isKubExporterReport(makeReport({ source: 'other' }))).toBe(false);
  });

  it('returns the CronJob owner reference when present', () => {
    const deployment = { apiVersion: 'apps/v1', kind: 'Deployment', name: 'deployment' };
    const cronJob = { apiVersion: 'batch/v1', kind: 'CronJob', name: 'backup' };

    expect(getCronJobOwner(makeReport({ metadata: { name: 'test-report', ownerReferences: [deployment, cronJob] } }))).toEqual(cronJob);
    expect(getCronJobOwner(makeReport())).toBeNull();
  });
});

describe('getReportLinks', () => {
  const annotation = 'kubexporter.bakito.ch/headlamp-link';

  it('parses and trims valid links from a JSON array while skipping invalid entries', () => {
    const links = getReportLinks(
      makeReport({
        metadata: {
          name: 'test-report',
          annotations: {
            [annotation]: JSON.stringify([
              { title: ' Dashboard ', url: ' https://example.com/dashboard ', icon: ' fa-solid fa-chart-line ' },
              { title: 'bad URL', url: 'javascript:alert(1)' },
              { title: ' ', url: 'https://example.com' },
              { title: 'bad icon', url: 'https://example.com', icon: 1 },
            ]),
          },
        },
      }),
    );

    expect(links).toEqual([{ title: 'Dashboard', url: 'https://example.com/dashboard', icon: 'fa-solid fa-chart-line' }]);
  });

  it('supports a legacy absolute HTTP(S) URL', () => {
    expect(
      getReportLinks(
        makeReport({ metadata: { name: 'test-report', annotations: { [annotation]: ' https://example.com/report ' } } }),
      ),
    ).toEqual([{ title: 'Open in Headlamp', url: 'https://example.com/report' }]);
  });

  it.each(['', 'not a URL', 'javascript:alert(1)', '/relative/path', '{invalid json}'])('returns no links for invalid value %s', (value) => {
    expect(getReportLinks(makeReport({ metadata: { name: 'test-report', annotations: { [annotation]: value } } }))).toEqual([]);
  });

  it('returns no links when the annotation is absent', () => {
    expect(getReportLinks(makeReport())).toEqual([]);
  });
});

describe('toIconifyIcon', () => {
  it('converts Font Awesome styles and passes through Iconify IDs', () => {
    expect(toIconifyIcon('fa-solid fa-chart-line')).toBe('fa6-solid:chart-line');
    expect(toIconifyIcon('fa-regular fa-calendar')).toBe('fa6-regular:calendar');
    expect(toIconifyIcon('fa-brands fa-github')).toBe('fa6-brands:github');
    expect(toIconifyIcon('mdi:home')).toBe('mdi:home');
  });

  it('uses the default icon when the value is missing or unusable', () => {
    expect(toIconifyIcon()).toBe('mdi:open-in-new');
    expect(toIconifyIcon('fa-solid')).toBe('mdi:open-in-new');
  });
});

describe('parseExportStats edge cases', () => {
  it('falls back to zero for non-numeric values and omits absent previous values', () => {
    const stats = parseExportStats(
      makeReport({
        results: [
          makeResult({ rule: 'resources', properties: { value: 'abc', prevValue: 'xyz' } }),
          makeResult({ rule: 'kinds', properties: { value: '5' } }),
          makeResult({ rule: 'unknown', properties: { value: '99' } }),
        ],
      }),
    );

    expect(stats).toEqual({ kinds: 5, resources: 0, prevResources: 0, size: 0, namespaces: 0, errors: 0, duration: '0s' });
  });

  it('returns defaults for a report without results', () => {
    expect(parseExportStats(makeReport())).toEqual({
      kinds: 0,
      resources: 0,
      size: 0,
      namespaces: 0,
      errors: 0,
      duration: '0s',
    });
  });
});

describe('parseResourceKindStats edge cases', () => {
  it('returns an empty list when there are no resource results', () => {
    expect(parseResourceKindStats(makeReport({ results: [makeResult()] }))).toEqual([]);
  });

  it('falls back to zero for non-numeric counters', () => {
    const [stats] = parseResourceKindStats(
      makeReport({
        results: [
          makeResult({
            policy: 'kubexporter/resource',
            rule: 'ConfigMap',
            properties: { instances: 'n/a', exportedInstances: '', exportedSizeBytes: '-' },
          }),
        ],
      }),
    );

    expect(stats.instances).toBe(0);
    expect(stats.exportedInstances).toBe(0);
    expect(stats.exportedSizeBytes).toBe(0);
  });
});

describe('formatBytes larger units', () => {
  it.each([
    [1536, '1.5 KB'],
    [1073741824, '1.0 GB'],
    [1099511627776, '1.0 TB'],
  ])('formats %i bytes', (bytes, expected) => {
    expect(formatBytes(bytes)).toBe(expected);
  });
});

describe('getReportLinks edge cases', () => {
  const annotation = 'kubexporter.bakito.ch/headlamp-link';
  const reportWith = (value: string) =>
    makeReport({ metadata: { name: 'test-report', annotations: { [annotation]: value } } });

  it('returns no links for an empty JSON array', () => {
    expect(getReportLinks(reportWith('[]'))).toEqual([]);
  });

  it.each(['{"title":"Dashboard","url":"https://example.com"}', '"https://example.com"', '42', 'null'])(
    'returns no links for non-array JSON %s',
    (value) => {
      expect(getReportLinks(reportWith(value))).toEqual([]);
    },
  );

  it('keeps multiple valid links in order and drops blank icons', () => {
    expect(
      getReportLinks(
        reportWith(
          JSON.stringify([
            { title: 'Grafana', url: 'https://grafana.example.com', icon: '  ' },
            { title: 'Logs', url: 'http://logs.example.com', icon: 'mdi:text' },
          ]),
        ),
      ),
    ).toEqual([
      { title: 'Grafana', url: 'https://grafana.example.com', icon: undefined },
      { title: 'Logs', url: 'http://logs.example.com', icon: 'mdi:text' },
    ]);
  });

  it.each([null, 'string', ['nested'], { title: 'no url' }, { url: 'https://example.com' }, { title: 'ftp', url: 'ftp://example.com' }])(
    'skips invalid entry %j',
    (entry) => {
      expect(getReportLinks(reportWith(JSON.stringify([entry])))).toEqual([]);
    },
  );

  it('supports a legacy plain HTTP URL', () => {
    expect(getReportLinks(reportWith('http://example.com'))).toEqual([{ title: 'Open in Headlamp', url: 'http://example.com' }]);
  });
});

describe('toIconifyIcon edge cases', () => {
  it.each([
    ['fas fa-house', 'fa6-solid:house'],
    ['far fa-bell', 'fa6-regular:bell'],
    ['fab fa-github', 'fa6-brands:github'],
    ['fa-chart-line', 'fa6-solid:chart-line'],
    ['  mdi:home  ', 'mdi:home'],
    ['   ', 'mdi:open-in-new'],
    ['chart-line', 'mdi:open-in-new'],
  ])('converts %s to %s', (input, expected) => {
    expect(toIconifyIcon(input)).toBe(expected);
  });
});

describe('getTrend', () => {
  it.each([
    [10, undefined, 'none'],
    [10, 5, 'up'],
    [5, 10, 'down'],
    [10, 10, 'same'],
  ] as const)('getTrend(%s, %s) returns %s', (current, previous, expected) => {
    expect(getTrend(current, previous)).toBe(expected);
  });
});