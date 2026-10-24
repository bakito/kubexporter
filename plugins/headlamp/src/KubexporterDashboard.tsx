import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Alert,
  AlertTitle,
  CircularProgress,
  Chip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  SelectChangeEvent,
  Card,
  CardContent,
} from '@mui/material';
import { Icon } from '@iconify/react';
import { ApiProxy } from '@kinvolk/headlamp-plugin/lib';
import StatsCards from './components/StatsCards';
import ResourceTable from './components/ResourceTable';
import { OpenReport } from './types';
import {
  parseExportStats,
  parseResourceKindStats,
  getLastExportTime,
  isKubexporterReport,
} from './utils';

// Custom Resource API types for OpenReports
const REPORT_GVK = {
  group: 'openreports.io',
  version: 'v1alpha1',
  kind: 'Report',
  plural: 'reports',
};

const CLUSTER_REPORT_GVK = {
  group: 'openreports.io',
  version: 'v1alpha1',
  kind: 'ClusterReport',
  plural: 'clusterreports',
};

const LABEL_SELECTOR = 'app.kubernetes.io/managed-by=kubexporter';

// Unique key of a report: names alone are not unique, as a ClusterReport and
// Reports in different namespaces may share the same name.
function reportKey(r: OpenReport): string {
  return `${r.kind}/${r.metadata.namespace || ''}/${r.metadata.name}`;
}

// Fetch CRs (cluster-wide / across all namespaces) of the currently selected
// cluster via Headlamp's API proxy. Returns an empty list if the CRD is not
// installed (404) or access is forbidden (403).
async function fetchCRs(gvk: typeof REPORT_GVK): Promise<OpenReport[]> {
  const url =
    `/apis/${gvk.group}/${gvk.version}/${gvk.plural}` +
    `?labelSelector=${encodeURIComponent(LABEL_SELECTOR)}`;

  let data: any;
  try {
    data = await ApiProxy.request(url);
  } catch (e: any) {
    if (e?.status === 404 || e?.status === 403) {
      return [];
    }
    throw e;
  }

  return (data?.items || [])
    .map((item: OpenReport) => ({ ...item, kind: item.kind || gvk.kind }))
    .filter((item: OpenReport) => item.source === 'kubexporter');
}

export default function KubexporterDashboard() {
  const [reports, setReports] = useState<OpenReport[]>([]);
  const [selectedReport, setSelectedReport] = useState<OpenReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const [clusterReports, namespacedReports] = await Promise.all([
          fetchCRs(CLUSTER_REPORT_GVK),
          fetchCRs(REPORT_GVK),
        ]);
        const all = [...clusterReports, ...namespacedReports];
        setReports(all);

        if (all.length > 0) {
          // Pick the most recent report
          const sorted = [...all].sort((a, b) => {
            const aTime = a.metadata.annotations?.['kubexporter.bakito.net/last-export'] || '';
            const bTime = b.metadata.annotations?.['kubexporter.bakito.net/last-export'] || '';
            return bTime.localeCompare(aTime);
          });
          setSelectedReport(sorted[0]);
        }
      } catch (e: any) {
        setError(e.message || 'Failed to load kubexporter reports');
      } finally {
        setLoading(false);
      }
    };

    load();
  }, []);

  const handleReportChange = (e: SelectChangeEvent) => {
    const key = e.target.value;
    const report = reports.find((r) => reportKey(r) === key);
    if (report) {
      setSelectedReport(report);
    }
  };

  if (loading) {
    return (
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          minHeight: 300,
          gap: 2,
        }}
      >
        <CircularProgress />
        <Typography color="text.secondary">
          Loading kubexporter reports...
        </Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        <AlertTitle>Error loading reports</AlertTitle>
        {error}
        <Typography variant="body2" sx={{ mt: 1 }}>
          Make sure the OpenReports CRD is installed and kubexporter has been
          run with the{' '}
          <Typography component="span" sx={{ fontFamily: 'monospace' }}>
            --openreports
          </Typography>{' '}
          flag.
        </Typography>
      </Alert>
    );
  }

  if (reports.length === 0) {
    return (
      <Alert severity="info" sx={{ mb: 2 }}>
        <AlertTitle>No kubexporter reports found</AlertTitle>
        No OpenReports Report or ClusterReport resources from kubexporter were
        found in this cluster.
        <Typography variant="body2" sx={{ mt: 1 }}>
          Run kubexporter with the{' '}
          <Typography component="span" sx={{ fontFamily: 'monospace' }}>
            --openreports
          </Typography>{' '}
          flag to create a report:
        </Typography>
        <Paper
          sx={{
            p: 2,
            mt: 1,
            backgroundColor: 'action.hover',
            fontFamily: 'monospace',
          }}
        >
          kubexporter --openreports --openreports-cluster-report
        </Paper>
      </Alert>
    );
  }

  if (!selectedReport) {
    return null;
  }

  const stats = parseExportStats(selectedReport);
  const resourceStats = parseResourceKindStats(selectedReport);
  const lastExport = getLastExportTime(selectedReport);

  return (
    <Box>
      {/* Header */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              flexWrap: 'wrap',
              gap: 2,
            }}
          >
            <Box>
              <Typography variant="h5" sx={{ fontWeight: 'bold', mb: 0.5 }}>
                Kubexporter Export Dashboard
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Chip
                  label={selectedReport.kind}
                  size="small"
                  sx={{ backgroundColor: '#e3f2fd', color: '#1565c0' }}
                />
                <Typography variant="body2" color="text.secondary">
                  {selectedReport.metadata.name}
                </Typography>
                {selectedReport.metadata.namespace && (
                  <Typography variant="body2" color="text.secondary">
                    in {selectedReport.metadata.namespace}
                  </Typography>
                )}
              </Box>
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              {lastExport && (
                <Box sx={{ textAlign: 'right' }}>
                  <Typography variant="caption" color="text.secondary">
                    Last export
                  </Typography>
                  <Typography variant="body2">
                    {new Date(lastExport).toLocaleString()}
                  </Typography>
                </Box>
              )}
              {reports.length > 1 && (
                <FormControl size="small" sx={{ minWidth: 200 }}>
                  <InputLabel>Report</InputLabel>
                  <Select
                    value={reportKey(selectedReport)}
                    label="Report"
                    onChange={handleReportChange}
                  >
                    {reports.map((r) => (
                      <MenuItem key={reportKey(r)} value={reportKey(r)}>
                        {r.metadata.namespace
                          ? `${r.metadata.namespace}/${r.metadata.name}`
                          : r.metadata.name}{' '}
                        ({r.kind})
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
            </Box>
          </Box>
        </CardContent>
      </Card>

      {/* Stats Cards */}
      {stats && <StatsCards stats={stats} />}

      {/* Resource Kinds Table */}
      {resourceStats.length > 0 && (
        <ResourceTable resources={resourceStats} />
      )}

      {/* Info footer */}
      <Box sx={{ mt: 3, mb: 1 }}>
        <Alert severity="info" variant="outlined">
          <Icon icon="mdi:information-outline" width={24} />
          <Typography variant="body2">
            This dashboard visualizes kubexporter export data stored as
            OpenReports CRDs. Run{' '}
            <Typography component="span" sx={{ fontFamily: 'monospace' }}>
              kubexporter --openreports
            </Typography>{' '}
            to create or update the report.
          </Typography>
        </Alert>
      </Box>
    </Box>
  );
}
