import {
  registerRoute,
  registerSidebarEntry,
} from '@kinvolk/headlamp-plugin/lib';
import KubexporterDashboard from './KubexporterDashboard';

// Register a top-level "Kubexporter" section in the cluster sidebar
registerSidebarEntry({
  parent: '',
  name: 'kubexporter',
  label: 'Kubexporter',
  url: '/kubexporter',
  icon: 'mdi:database-export',
});

// Register the dashboard entry inside the "Kubexporter" section
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
