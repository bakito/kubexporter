import React, { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  TablePagination,
  Paper,
  Tooltip,
} from '@mui/material';
import { Icon } from '@iconify/react';
import { ResourceKindStats } from '../types';
import { formatBytes, formatNumber, getTrend } from '../utils';

type SortDirection = 'asc' | 'desc';

// Renders a trend arrow icon next to a value
function TrendIcon({ current, previous }: { current: number; previous?: number }) {
  const trend = getTrend(current, previous);
  if (trend === 'none') return null;

  let icon: string;
  let color: string;
  switch (trend) {
    case 'up':
      icon = 'mdi:arrow-up';
      color = '#388e3c';
      break;
    case 'down':
      icon = 'mdi:arrow-down';
      color = '#d32f2f';
      break;
    default:
      icon = 'mdi:minus';
      color = '#757575';
  }

  const iconEl = (
    <Box component="span" sx={{ ml: 0.5, color, verticalAlign: 'middle' }}>
      <Icon icon={icon} width={14} />
    </Box>
  );

  if (previous !== undefined) {
    return (
      <Tooltip title={`Previous: ${previous.toLocaleString('en-US')}`}>
        {iconEl}
      </Tooltip>
    );
  }

  return iconEl;
}

interface ResourceTableProps {
  resources: ResourceKindStats[];
}

// Split "group.Kind" (e.g. "apps.Deployment") into kind and group; core kinds have no group.
function splitGroupKind(groupKind: string): { kind: string; group: string } {
  const idx = groupKind.lastIndexOf('.');
  return idx < 0
    ? { kind: groupKind, group: '' }
    : { kind: groupKind.substring(idx + 1), group: groupKind.substring(0, idx) };
}

// Compare by kind name first, then by group.
function compareGroupKind(a: string, b: string): number {
  const ga = splitGroupKind(a);
  const gb = splitGroupKind(b);
  return ga.kind.localeCompare(gb.kind) || ga.group.localeCompare(gb.group);
}

export default function ResourceTable({ resources }: ResourceTableProps) {
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [sortField, setSortField] = useState<keyof ResourceKindStats>('groupKind');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  const handleSort = (field: keyof ResourceKindStats) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const sortedResources = [...resources].sort((a, b) => {
    if (sortField === 'groupKind') {
      const cmp = compareGroupKind(a.groupKind, b.groupKind);
      return sortDirection === 'asc' ? cmp : -cmp;
    }

    const aVal = a[sortField];
    const bVal = b[sortField];

    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return sortDirection === 'asc'
        ? aVal.localeCompare(bVal)
        : bVal.localeCompare(aVal);
    }

    if (typeof aVal === 'number' && typeof bVal === 'number') {
      return sortDirection === 'asc' ? aVal - bVal : bVal - aVal;
    }

    return 0;
  });

  const pagedResources = sortedResources.slice(
    page * rowsPerPage,
    page * rowsPerPage + rowsPerPage,
  );

  const SortableHeader = (field: keyof ResourceKindStats, label: string) => (
    <TableCell>
      <TableSortLabel
        active={sortField === field}
        direction={sortField === field ? sortDirection : 'asc'}
        onClick={() => handleSort(field)}
      >
        {label}
      </TableSortLabel>
    </TableCell>
  );

  return (
    <Card>
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <Box sx={{ mr: 1, color: 'text.secondary', display: 'flex' }}>
            <Icon icon="mdi:table" width={24} />
          </Box>
          <Typography variant="h6" component="h3">
            Resource Kinds ({resources.length})
          </Typography>
        </Box>
        <TableContainer component={Paper} sx={{ boxShadow: 'none' }}>
          <Table size="small">
            <TableHead>
              <TableRow sx={{ backgroundColor: 'action.hover' }}>
                {SortableHeader('groupKind', 'Kind')}
                {SortableHeader('instances', 'Instances')}
                {SortableHeader('exportedInstances', 'Exported')}
                {SortableHeader('pages', 'Pages')}
                {SortableHeader('exportedSizeBytes', 'Size')}
                {SortableHeader('queryDuration', 'Query Time')}
                {SortableHeader('exportDuration', 'Export Time')}
              </TableRow>
            </TableHead>
            <TableBody>
              {pagedResources.map((res, idx) => (
                <TableRow
                  key={res.groupKind}
                  hover
                  sx={{
                    backgroundColor:
                      idx % 2 === 0 ? 'background.default' : 'action.hover',
                  }}
                >
                  <TableCell>
                    <Typography variant="body2" sx={{ fontWeight: 'medium' }}>
                      {res.groupKind}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'flex-end' }}>
                      {formatNumber(res.instances)}
                      <TrendIcon current={res.instances} previous={res.prevInstances} />
                    </Box>
                  </TableCell>
                  <TableCell align="right">
                    <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'flex-end' }}>
                      {formatNumber(res.exportedInstances)}
                      <TrendIcon current={res.exportedInstances} previous={res.prevExportedInstances} />
                    </Box>
                  </TableCell>
                  <TableCell align="right">{res.pages}</TableCell>
                  <TableCell align="right">
                    <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'flex-end' }}>
                      {formatBytes(res.exportedSizeBytes)}
                      <TrendIcon current={res.exportedSizeBytes} previous={res.prevExportedSizeBytes} />
                    </Box>
                  </TableCell>
                  <TableCell>{res.queryDuration}</TableCell>
                  <TableCell>{res.exportDuration}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
        <TablePagination
          component="div"
          count={resources.length}
          page={page}
          onPageChange={(_, newPage) => setPage(newPage)}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(e) => {
            setRowsPerPage(parseInt(e.target.value, 10));
            setPage(0);
          }}
          rowsPerPageOptions={[10, 25, 50, 100]}
        />
      </CardContent>
    </Card>
  );
}
