import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { MissionUiProvider } from './context/MissionUiContext';
import { WorkspaceShellLayout } from './components/workspace/WorkspaceShellLayout';
import { WorkspaceDashboardView } from './components/dashboard/WorkspaceDashboardView';
import { ActivityLogPage } from './routes/ActivityLogPage';
import { AutopilotPlaceholderPage } from './routes/AutopilotPlaceholderPage';
import { MissionApprovalsPage } from './routes/MissionApprovalsPage';
import { MissionAgentsPage } from './routes/MissionAgentsPage';
import { MissionDocsPage } from './routes/MissionDocsPage';
import { MissionFactoryPage } from './routes/MissionFactoryPage';
import { MissionFeedPage } from './routes/MissionFeedPage';
import { MissionFeedbackPage } from './routes/MissionFeedbackPage';
import { MissionCalendarPage } from './routes/MissionCalendarPage';
import { MissionCouncilPage } from './routes/MissionCouncilPage';
import { MissionMemoryPage } from './routes/MissionMemoryPage';
import { MissionRadarPage } from './routes/MissionRadarPage';
import { MissionProjectsPage } from './routes/MissionProjectsPage';
import { MissionSystemPage } from './routes/MissionSystemPage';
import { MissionTeamPage } from './routes/MissionTeamPage';
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
              <Route path="approvals" element={<MissionApprovalsPage />} />
              <Route path="council" element={<MissionCouncilPage />} />
              <Route path="calendar" element={<MissionCalendarPage />} />
              <Route path="projects" element={<MissionProjectsPage />} />
              <Route path="memory" element={<MissionMemoryPage />} />
              <Route path="docs" element={<MissionDocsPage />} />
              <Route path="people" element={<WorkspaceModulePlaceholder segment="people" />} />
              <Route path="office" element={<WorkspaceModulePlaceholder segment="office" />} />
              <Route path="team" element={<MissionTeamPage />} />
              <Route path="system" element={<MissionSystemPage />} />
              <Route path="radar" element={<MissionRadarPage />} />
              <Route path="factory" element={<MissionFactoryPage />} />
              <Route path="pipeline" element={<WorkspaceModulePlaceholder segment="pipeline" />} />
              <Route path="feedback" element={<MissionFeedbackPage />} />
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
