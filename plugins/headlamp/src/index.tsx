import {
  registerRoute,
  registerSidebarEntry,
} from '@kinvolk/headlamp-plugin/lib';
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
