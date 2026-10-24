import React from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Card,
  CardContent,
} from '@mui/material';
import { Icon } from '@iconify/react';
import { ExportStats } from '../types';
import { formatBytes, formatNumber } from '../utils';

interface StatsCardsProps {
  stats: ExportStats;
}

interface StatCardProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  color: string;
}

function StatCard({ icon, label, value, color }: StatCardProps) {
  return (
    <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          <Box
            sx={{
              color,
              mr: 1,
              display: 'flex',
              alignItems: 'center',
            }}
          >
            {icon}
          </Box>
          <Typography variant="subtitle2" color="text.secondary">
            {label}
          </Typography>
        </Box>
        <Typography variant="h4" sx={{ fontWeight: 'bold', color }}>
          {value}
        </Typography>
      </CardContent>
    </Card>
  );
}

export default function StatsCards({ stats }: StatsCardsProps) {
  return (
    <Grid container spacing={2} sx={{ mb: 3 }}>
      <Grid item xs={12} sm={6} md={4} lg={2}>
        <StatCard
          icon={<Icon icon="mdi:database-outline" width={24} />}
          label="Resources"
          value={formatNumber(stats.resources)}
          color="#1976d2"
        />
      </Grid>
      <Grid item xs={12} sm={6} md={4} lg={2}>
        <StatCard
          icon={<Icon icon="mdi:layers-outline" width={24} />}
          label="Kinds"
          value={formatNumber(stats.kinds)}
          color="#388e3c"
        />
      </Grid>
      <Grid item xs={12} sm={6} md={4} lg={2}>
        <StatCard
          icon={<Icon icon="mdi:cloud-outline" width={24} />}
          label="Namespaces"
          value={formatNumber(stats.namespaces)}
          color="#f57c00"
        />
      </Grid>
      <Grid item xs={12} sm={6} md={4} lg={2}>
        <StatCard
          icon={<Icon icon="mdi:speedometer" width={24} />}
          label="Size"
          value={formatBytes(stats.size)}
          color="#7b1fa2"
        />
      </Grid>
      <Grid item xs={12} sm={6} md={4} lg={2}>
        <StatCard
          icon={<Icon icon="mdi:clock-outline" width={24} />}
          label="Duration"
          value={stats.duration}
          color="#00838f"
        />
      </Grid>
      <Grid item xs={12} sm={6} md={4} lg={2}>
        <StatCard
          icon={<Icon icon="mdi:bug-outline" width={24} />}
          label="Errors"
          value={formatNumber(stats.errors)}
          color={stats.errors > 0 ? '#d32f2f' : '#388e3c'}
        />
      </Grid>
    </Grid>
  );
}
