import { MissionUiProvider, useMissionUi } from './context/MissionUiContext';
import { WorkspaceDashboardView } from './components/dashboard/WorkspaceDashboardView';
import { MissionWorkspacePage } from './components/workspace/MissionWorkspacePage';

function Routes() {
  const { activeWorkspace } = useMissionUi();
  return activeWorkspace ? <MissionWorkspacePage /> : <WorkspaceDashboardView />;
}

export default function App() {
  return (
    <MissionUiProvider>
      <Routes />
    </MissionUiProvider>
  );
}
