import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { MissionUiProvider } from './context/MissionUiContext';
import { WorkspaceShellLayout } from './components/workspace/WorkspaceShellLayout';
import { WorkspaceDashboardView } from './components/dashboard/WorkspaceDashboardView';
import { ActivityLogPage } from './routes/ActivityLogPage';
import { AutopilotPlaceholderPage } from './routes/AutopilotPlaceholderPage';
import { MissionAgentsPage } from './routes/MissionAgentsPage';
import { MissionFeedPage } from './routes/MissionFeedPage';
import { MissionProjectsPage } from './routes/MissionProjectsPage';
import { MissionTasksPage } from './routes/MissionTasksPage';
import { NotFoundPage } from './routes/NotFoundPage';
import { ProductWorkspaceOutlet } from './routes/ProductWorkspaceOutlet';
import { WorkspaceModulePlaceholder } from './routes/WorkspaceModulePlaceholder';

export default function App() {
  return (
    <BrowserRouter>
      <MissionUiProvider>
        <Routes>
          <Route path="/" element={<WorkspaceDashboardView />} />
          <Route path="/p/:productId" element={<ProductWorkspaceOutlet />}>
            <Route index element={<Navigate to="tasks" replace />} />
            <Route element={<WorkspaceShellLayout />}>
              <Route path="tasks" element={<MissionTasksPage />} />
              <Route path="agents" element={<MissionAgentsPage />} />
              <Route path="feed" element={<MissionFeedPage />} />
              <Route path="content" element={<WorkspaceModulePlaceholder segment="content" />} />
              <Route path="approvals" element={<WorkspaceModulePlaceholder segment="approvals" />} />
              <Route path="council" element={<WorkspaceModulePlaceholder segment="council" />} />
              <Route path="calendar" element={<WorkspaceModulePlaceholder segment="calendar" />} />
              <Route path="projects" element={<MissionProjectsPage />} />
              <Route path="memory" element={<WorkspaceModulePlaceholder segment="memory" />} />
              <Route path="docs" element={<WorkspaceModulePlaceholder segment="docs" />} />
              <Route path="people" element={<WorkspaceModulePlaceholder segment="people" />} />
              <Route path="office" element={<WorkspaceModulePlaceholder segment="office" />} />
              <Route path="team" element={<WorkspaceModulePlaceholder segment="team" />} />
              <Route path="system" element={<WorkspaceModulePlaceholder segment="system" />} />
              <Route path="radar" element={<WorkspaceModulePlaceholder segment="radar" />} />
              <Route path="factory" element={<WorkspaceModulePlaceholder segment="factory" />} />
              <Route path="pipeline" element={<WorkspaceModulePlaceholder segment="pipeline" />} />
              <Route path="feedback" element={<WorkspaceModulePlaceholder segment="feedback" />} />
            </Route>
          </Route>
          <Route path="/autopilot" element={<AutopilotPlaceholderPage />} />
          <Route path="/activity" element={<ActivityLogPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </MissionUiProvider>
    </BrowserRouter>
  );
}
