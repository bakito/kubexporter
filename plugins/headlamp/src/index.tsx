import React from 'react';
import { useHistory } from 'react-router-dom';
import {
  registerRoute,
  registerSidebarEntry,
  registerDetailsViewHeaderAction,
  registerResourceTableColumnsProcessor,
  Router,
} from '@kinvolk/headlamp-plugin/lib';
import { ActionButton } from '@kinvolk/headlamp-plugin/lib/CommonComponents';
import KubexporterDashboard from './KubexporterDashboard';

// Register a top-level "KubExporter" section in the cluster sidebar
registerSidebarEntry({
  parent: '',
  name: 'kubexporter',
  label: 'KubExporter',
  url: '/kubexporter',
  icon: 'mdi:database-export',
});

// Register the dashboard entry inside the "KubExporter" section
registerSidebarEntry({
  parent: 'kubexporter',
  name: 'kubexporter-dashboard',
  label: 'Dashboard',
  url: '/kubexporter',
});

// Register the route for the dashboard page
registerRoute({
  path: '/kubexporter',
  name: 'kubexporter-dashboard',
  component: KubexporterDashboard,
  sidebar: 'kubexporter-dashboard',
  exact: true,
});

// Check if a resource is an OpenReports Report created by kubexporter
function isKubexporterReportResource(resource: any): boolean {
  const data = resource?.jsonData ?? resource;
  if (!data || data.kind !== 'Report') {
    return false;
  }
  const apiVersion: string = data.apiVersion || '';
  return apiVersion.startsWith('openreports.io/') && data.source === 'kubexporter';
}

function DashboardButton({ item, buttonStyle }: { item: any; buttonStyle?: 'action' | 'menu' }) {
  const history = useHistory();
  if (!isKubexporterReportResource(item)) {
    return null;
  }
  return (
    <ActionButton
      description="View KubExporter Dashboard"
      icon="mdi:chart-box"
      buttonStyle={buttonStyle}
      onClick={() => history.push(Router.createRouteURL('kubexporter-dashboard'))}
    />
  );
}

// Add a header action linking to the dashboard in the details view of kubexporter Reports
registerDetailsViewHeaderAction(({ item }: { item: any }) => <DashboardButton item={item} />);

// Add a dashboard link column to the Reports list table
registerResourceTableColumnsProcessor(({ id, columns }) => {
  if (id !== 'headlamp-reports') {
    return columns;
  }
  return [
    ...columns,
    {
      id: 'kubexporter-dashboard',
      label: 'Dashboard',
      gridTemplate: 'min-content',
      disableFiltering: true,
      getValue: () => null,
      render: (item: any) => <DashboardButton item={item} />,
    },
  ];
});
