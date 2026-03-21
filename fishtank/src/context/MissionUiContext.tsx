import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useState,
  type ReactNode,
} from 'react';
import { ArmsClient } from '../api/armsClient';
import { readArmsEnv } from '../config/armsEnv';
import { useArmsLiveFeed } from '../hooks/useArmsLiveFeed';
import type { Agent, FeedEvent, Task, WorkspaceStats } from '../domain/types';
import {
  agentHealthToAgent,
  apiProductToWorkspaceStats,
  apiTaskToTask,
  summarizeAgentCounts,
  summarizeTaskCounts,
} from '../mappers/missionMappers';

type LiveAction = { kind: 'append'; event: FeedEvent } | { kind: 'clear' };

function liveEventsReducer(state: FeedEvent[], action: LiveAction): FeedEvent[] {
  const cap = 200;
  switch (action.kind) {
    case 'append': {
      const next = [action.event, ...state];
      return next.slice(0, cap);
    }
    case 'clear':
      return [];
    default:
      return state;
  }
}

export interface MissionUiValue {
  workspaces: WorkspaceStats[];
  activeWorkspace: WorkspaceStats | null;
  tasks: Task[];
  agents: Agent[];
  events: FeedEvent[];
  isOnline: boolean;
  boardLoading: boolean;
  listLoading: boolean;
  apiError: string | null;
  goHome: () => void;
  openWorkspace: (workspace: WorkspaceStats) => Promise<void>;
  refreshWorkspaces: () => Promise<void>;
  registerProduct: (name: string, workspaceId: string) => Promise<void>;
  dismissError: () => void;
}

const MissionUiContext = createContext<MissionUiValue | null>(null);

async function loadWorkspaceSummaries(client: ArmsClient): Promise<WorkspaceStats[]> {
  const products = await client.listProducts();
  return Promise.all(
    products.map(async (p) => {
      const tasks = await client.listProductTasks(p.id);
      let health: Awaited<ReturnType<ArmsClient['listProductAgentHealth']>> = [];
      try {
        health = await client.listProductAgentHealth(p.id);
      } catch {
        health = [];
      }
      return apiProductToWorkspaceStats(p, summarizeTaskCounts(tasks), summarizeAgentCounts(health));
    }),
  );
}

export function MissionUiProvider({ children }: { children: ReactNode }) {
  const env = useMemo(() => readArmsEnv(), []);
  const client = useMemo(() => new ArmsClient(env), [env]);

  const [workspaces, setWorkspaces] = useState<WorkspaceStats[]>([]);
  const [activeWorkspace, setActiveWorkspace] = useState<WorkspaceStats | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [isOnline, setIsOnline] = useState(false);
  const [boardLoading, setBoardLoading] = useState(false);
  const [listLoading, setListLoading] = useState(true);
  const [apiError, setApiError] = useState<string | null>(null);

  const [liveEvents, dispatchLive] = useReducer(liveEventsReducer, []);

  const appendLive = useCallback((event: FeedEvent) => {
    dispatchLive({ kind: 'append', event });
  }, []);

  useArmsLiveFeed(activeWorkspace?.id ?? null, env, appendLive);

  useEffect(() => {
    dispatchLive({ kind: 'clear' });
  }, [activeWorkspace?.id]);

  const refreshWorkspaces = useCallback(async () => {
    setListLoading(true);
    setApiError(null);
    try {
      await client.health();
      setIsOnline(true);
      const next = await loadWorkspaceSummaries(client);
      setWorkspaces(next);
    } catch {
      setIsOnline(false);
      setWorkspaces([]);
      setApiError('Cannot load products. Is arms running and CORS configured? (ARMS_CORS_ALLOW_ORIGIN)');
    } finally {
      setListLoading(false);
    }
  }, [client]);

  const pingOnce = useCallback(async () => {
    try {
      await client.health();
      setIsOnline(true);
    } catch {
      setIsOnline(false);
    }
  }, [client]);

  useEffect(() => {
    void refreshWorkspaces();
  }, [refreshWorkspaces]);

  useEffect(() => {
    const id = window.setInterval(() => void pingOnce(), 30_000);
    return () => window.clearInterval(id);
  }, [pingOnce]);

  const goHome = useCallback(() => {
    setActiveWorkspace(null);
    setTasks([]);
    setAgents([]);
  }, []);

  const openWorkspace = useCallback(
    async (workspace: WorkspaceStats) => {
      setActiveWorkspace(workspace);
      setBoardLoading(true);
      setApiError(null);
      try {
        const [apiTasks, health] = await Promise.all([
          client.listProductTasks(workspace.id),
          client.listProductAgentHealth(workspace.id).catch(() => []),
        ]);
        setTasks(apiTasks.map(apiTaskToTask));
        setAgents(health.map(agentHealthToAgent));
      } catch {
        setTasks([]);
        setAgents([]);
        setApiError('Failed to load tasks or agent health for this product.');
      } finally {
        setBoardLoading(false);
      }
    },
    [client],
  );

  const registerProduct = useCallback(
    async (name: string, workspaceId: string) => {
      setApiError(null);
      await client.createProduct({ name: name.trim(), workspace_id: workspaceId.trim() });
      await refreshWorkspaces();
    },
    [client, refreshWorkspaces],
  );

  const dismissError = useCallback(() => setApiError(null), []);

  const value = useMemo<MissionUiValue>(
    () => ({
      workspaces,
      activeWorkspace,
      tasks,
      agents,
      events: liveEvents,
      isOnline,
      boardLoading,
      listLoading,
      apiError,
      goHome,
      openWorkspace,
      refreshWorkspaces,
      registerProduct,
      dismissError,
    }),
    [
      workspaces,
      activeWorkspace,
      tasks,
      agents,
      liveEvents,
      isOnline,
      boardLoading,
      listLoading,
      apiError,
      goHome,
      openWorkspace,
      refreshWorkspaces,
      registerProduct,
      dismissError,
    ],
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
