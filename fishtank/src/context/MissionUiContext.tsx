import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import type { Agent, FeedEvent, Task, WorkspaceStats } from '../domain/types';
import { MOCK_AGENTS, MOCK_EVENTS, MOCK_TASKS, MOCK_WORKSPACES } from '../data/mock';

export interface MissionUiValue {
  workspaces: WorkspaceStats[];
  activeWorkspace: WorkspaceStats | null;
  tasks: Task[];
  agents: Agent[];
  events: FeedEvent[];
  isOnline: boolean;
  goHome: () => void;
  openWorkspace: (workspace: WorkspaceStats) => void;
}

const MissionUiContext = createContext<MissionUiValue | null>(null);

export function MissionUiProvider({ children }: { children: ReactNode }) {
  const [workspaces] = useState<WorkspaceStats[]>(MOCK_WORKSPACES);
  const [activeWorkspace, setActiveWorkspace] = useState<WorkspaceStats | null>(null);
  const [tasks] = useState<Task[]>(MOCK_TASKS);
  const [agents] = useState<Agent[]>(MOCK_AGENTS);
  const [events] = useState<FeedEvent[]>(MOCK_EVENTS);
  const [isOnline] = useState(true);

  const goHome = useCallback(() => setActiveWorkspace(null), []);
  const openWorkspace = useCallback((workspace: WorkspaceStats) => setActiveWorkspace(workspace), []);

  const value = useMemo<MissionUiValue>(
    () => ({
      workspaces,
      activeWorkspace,
      tasks,
      agents,
      events,
      isOnline,
      goHome,
      openWorkspace,
    }),
    [workspaces, activeWorkspace, tasks, agents, events, isOnline, goHome, openWorkspace],
  );

  return <MissionUiContext.Provider value={value}>{children}</MissionUiContext.Provider>;
}

export function useMissionUi(): MissionUiValue {
  const ctx = useContext(MissionUiContext);
  if (!ctx) {
    throw new Error('useMissionUi must be used within MissionUiProvider');
  }
  return ctx;
}
