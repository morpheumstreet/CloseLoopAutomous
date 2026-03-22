import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { MissionUiProvider } from './context/MissionUiContext';
import { WorkspaceDashboardView } from './components/dashboard/WorkspaceDashboardView';
import { ActivityLogPage } from './routes/ActivityLogPage';
import { AutopilotPlaceholderPage } from './routes/AutopilotPlaceholderPage';
import { NotFoundPage } from './routes/NotFoundPage';
import { ProductWorkspaceOutlet } from './routes/ProductWorkspaceOutlet';

export default function App() {
  return (
    <BrowserRouter>
      <MissionUiProvider>
        <Routes>
          <Route path="/" element={<WorkspaceDashboardView />} />
          <Route path="/p/:productId" element={<ProductWorkspaceOutlet />} />
          <Route path="/autopilot" element={<AutopilotPlaceholderPage />} />
          <Route path="/activity" element={<ActivityLogPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </MissionUiProvider>
    </BrowserRouter>
  );
}
