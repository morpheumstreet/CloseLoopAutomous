import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { ArmsClient, type OperationsLogQuery } from '../api/armsClient';
import type { ApiProductDetail, ApiTask, ApiVersion } from '../api/armsTypes';
import type { ArmsEnv } from '../config/armsEnv';
import { readArmsEnv } from '../config/armsEnv';
import type { StalledTaskRow, Task, TaskStatus } from '../domain/types';
import { useArmsLiveFeed } from '../hooks/useArmsLiveFeed';
import type { Agent, FeedEvent, WorkspaceStats } from '../domain/types';
import { shouldRefreshBoardOnArmsSseType } from '../lib/liveBoardRefresh';
import {
  agentHealthToAgent,
  apiProductToWorkspaceStats,
  apiTaskToTask,
  stalledApiToRows,
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

function mergeTaskIntoList(list: Task[], updated: ApiTask): Task[] {
  const mapped = apiTaskToTask(updated);
  const idx = list.findIndex((t) => t.id === mapped.id);
  if (idx < 0) return [mapped, ...list];
  const next = [...list];
  next[idx] = mapped;
  return next;
}

export interface MissionUiValue {
  armsEnv: ArmsEnv;
  client: ArmsClient;
  workspaces: WorkspaceStats[];
  activeWorkspace: WorkspaceStats | null;
  productDetail: ApiProductDetail | null;
  stalledTasks: StalledTaskRow[];
  tasks: Task[];
  agents: Agent[];
  events: FeedEvent[];
  isOnline: boolean;
  feedLive: boolean;
  bumpFeedReconnect: () => void;
  boardLoading: boolean;
  boardLoadFailed: boolean;
  listLoading: boolean;
  apiError: string | null;
  goHome: () => void;
  openWorkspace: (workspace: WorkspaceStats) => Promise<void>;
  refreshWorkspaces: () => Promise<void>;
  refreshActiveBoard: (opts?: { silent?: boolean }) => Promise<void>;
  registerProduct: (name: string, workspaceId: string) => Promise<string>;
  dismissError: () => void;
  fetchVersion: () => Promise<ApiVersion>;
  fetchOperationsLog: (q?: OperationsLogQuery) => ReturnType<ArmsClient['listOperationsLog']>;
  patchTaskStatus: (taskId: string, status: TaskStatus) => Promise<void>;
  createTaskForProduct: (ideaId: string | null, spec: string, newIdeaId?: string | null) => Promise<void>;
  approveTaskPlan: (taskId: string) => Promise<void>;
  rejectTaskPlan: (taskId: string, statusReason?: string) => Promise<void>;
  saveTaskClarifications: (taskId: string, clarificationsJson: string) => Promise<void>;
  dispatchTaskById: (taskId: string, estimatedCost?: number) => Promise<void>;
  completeTaskById: (taskId: string) => Promise<void>;
  stallNudgeTask: (taskId: string, note?: string) => Promise<void>;
  openTaskPullRequest: (taskId: string, headBranch: string, title?: string) => Promise<void>;
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
  const [productDetail, setProductDetail] = useState<ApiProductDetail | null>(null);
  const [stalledTasks, setStalledTasks] = useState<StalledTaskRow[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [isOnline, setIsOnline] = useState(false);
  const [feedLive, setFeedLive] = useState(false);
  const [feedEpoch, setFeedEpoch] = useState(0);
  const [boardLoading, setBoardLoading] = useState(false);
  const [boardLoadFailed, setBoardLoadFailed] = useState(false);
  const [listLoading, setListLoading] = useState(true);
  const [apiError, setApiError] = useState<string | null>(null);

  const [liveEvents, dispatchLive] = useReducer(liveEventsReducer, []);

  const boardRefreshDebounceRef = useRef<ReturnType<typeof window.setTimeout> | null>(null);
  const refreshActiveBoardRef = useRef<(opts?: { silent?: boolean }) => Promise<void>>(async () => {});

  const flushBoardRefreshTimer = useCallback(() => {
    if (boardRefreshDebounceRef.current !== null) {
      window.clearTimeout(boardRefreshDebounceRef.current);
      boardRefreshDebounceRef.current = null;
    }
  }, []);

  const onFeedLive = useCallback((live: boolean) => {
    setFeedLive(live);
  }, []);

  useEffect(() => {
    dispatchLive({ kind: 'clear' });
  }, [activeWorkspace?.id]);

  useEffect(() => {
    flushBoardRefreshTimer();
  }, [activeWorkspace?.id, flushBoardRefreshTimer]);

  useEffect(
    () => () => {
      flushBoardRefreshTimer();
    },
    [flushBoardRefreshTimer],
  );

  const bumpFeedReconnect = useCallback(() => {
    setFeedEpoch((e) => e + 1);
  }, []);

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
    const ms = isOnline ? 30_000 : 8_000;
    const id = window.setInterval(() => void pingOnce(), ms);
    return () => window.clearInterval(id);
  }, [pingOnce, isOnline]);

  useEffect(() => {
    const onBrowserOnline = () => void refreshWorkspaces();
    const onVisible = () => {
      if (document.visibilityState === 'visible') void pingOnce();
    };
    window.addEventListener('online', onBrowserOnline);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      window.removeEventListener('online', onBrowserOnline);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [pingOnce, refreshWorkspaces]);

  const goHome = useCallback(() => {
    setActiveWorkspace(null);
    setProductDetail(null);
    setStalledTasks([]);
    setTasks([]);
    setAgents([]);
    setBoardLoadFailed(false);
  }, []);

  const refreshActiveBoard = useCallback(
    async (opts?: { silent?: boolean }) => {
      if (!activeWorkspace) return;
      const wid = activeWorkspace.id;
      const loud = !opts?.silent;
      if (loud) {
        setBoardLoading(true);
        setBoardLoadFailed(false);
      }
      try {
        const [apiTasks, stalledRaw, detail, health] = await Promise.all([
          client.listProductTasks(wid),
          client.listStalledTasks(wid),
          client.getProduct(wid).catch(() => null),
          client.listProductAgentHealth(wid).catch(() => []),
        ]);
        setTasks(apiTasks.map(apiTaskToTask));
        setStalledTasks(stalledApiToRows(stalledRaw));
        setAgents(health.map(agentHealthToAgent));
        if (detail) setProductDetail(detail);
        const counts = summarizeTaskCounts(apiTasks);
        const agentCounts = summarizeAgentCounts(health);
        setWorkspaces((prev) => prev.map((w) => (w.id === wid ? { ...w, taskCounts: counts, agentCounts } : w)));
        setActiveWorkspace((prev) => {
          if (!prev || prev.id !== wid) return prev;
          if (detail) return apiProductToWorkspaceStats(detail, counts, agentCounts);
          return { ...prev, taskCounts: counts, agentCounts };
        });
      } catch {
        if (loud) {
          setBoardLoadFailed(true);
          setApiError('Failed to refresh the board for this product.');
        }
      } finally {
        if (loud) setBoardLoading(false);
      }
    },
    [activeWorkspace, client],
  );

  refreshActiveBoardRef.current = refreshActiveBoard;

  const appendLive = useCallback(
    (event: FeedEvent) => {
      dispatchLive({ kind: 'append', event });
      const t = event.armsType;
      if (t && shouldRefreshBoardOnArmsSseType(t)) {
        if (boardRefreshDebounceRef.current !== null) {
          window.clearTimeout(boardRefreshDebounceRef.current);
        }
        boardRefreshDebounceRef.current = window.setTimeout(() => {
          boardRefreshDebounceRef.current = null;
          void refreshActiveBoardRef.current({ silent: true });
        }, 400);
      }
    },
    [],
  );

  useArmsLiveFeed(activeWorkspace?.id ?? null, env, appendLive, {
    reconnectEpoch: feedEpoch,
    onConnectionLive: onFeedLive,
  });

  const openWorkspace = useCallback(
    async (workspace: WorkspaceStats) => {
      setActiveWorkspace(workspace);
      setBoardLoading(true);
      setBoardLoadFailed(false);
      setApiError(null);
      setProductDetail(null);
      try {
        const [detail, apiTasks, health, stalledRaw] = await Promise.all([
          client.getProduct(workspace.id).catch(() => null),
          client.listProductTasks(workspace.id),
          client.listProductAgentHealth(workspace.id).catch(() => []),
          client.listStalledTasks(workspace.id),
        ]);
        const tc = summarizeTaskCounts(apiTasks);
        const ac = summarizeAgentCounts(health);
        if (detail) {
          setProductDetail(detail);
          setActiveWorkspace(apiProductToWorkspaceStats(detail, tc, ac));
        } else {
          setActiveWorkspace((prev) =>
            prev && prev.id === workspace.id ? { ...prev, taskCounts: tc, agentCounts: ac } : prev,
          );
        }
        setTasks(apiTasks.map(apiTaskToTask));
        setAgents(health.map(agentHealthToAgent));
        setStalledTasks(stalledApiToRows(stalledRaw));
      } catch {
        setTasks([]);
        setAgents([]);
        setStalledTasks([]);
        setBoardLoadFailed(true);
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
      const p = await client.createProduct({ name: name.trim(), workspace_id: workspaceId.trim() });
      await refreshWorkspaces();
      return p.id;
    },
    [client, refreshWorkspaces],
  );

  const dismissError = useCallback(() => setApiError(null), []);

  const fetchVersion = useCallback(() => client.version(), [client]);

  const fetchOperationsLog = useCallback((q?: OperationsLogQuery) => client.listOperationsLog(q ?? {}), [client]);

  const patchTaskStatus = useCallback(
    async (taskId: string, status: TaskStatus) => {
      const updated = await client.patchTask(taskId, { status });
      setTasks((prev) => mergeTaskIntoList(prev, updated));
    },
    [client],
  );

  const createTaskForProduct = useCallback(
    async (ideaId: string | null, spec: string, newIdeaId?: string | null) => {
      if (!activeWorkspace) {
        throw new Error('No active workspace');
      }
      const s = spec.trim();
      const nid = newIdeaId?.trim() ?? '';
      if (nid !== '') {
        await client.createTask({ product_id: activeWorkspace.id, spec: s, new_idea_id: nid });
      } else if (ideaId == null || ideaId.trim() === '') {
        await client.createTask({ product_id: activeWorkspace.id, spec: s });
      } else {
        await client.createTask({ idea_id: ideaId.trim(), spec: s });
      }
      await refreshActiveBoard();
    },
    [activeWorkspace, client, refreshActiveBoard],
  );

  const approveTaskPlan = useCallback(
    async (taskId: string) => {
      const updated = await client.approvePlan(taskId, {});
      setTasks((prev) => mergeTaskIntoList(prev, updated));
      await refreshActiveBoard();
    },
    [client, refreshActiveBoard],
  );

  const rejectTaskPlan = useCallback(
    async (taskId: string, statusReason?: string) => {
      const body = statusReason?.trim() ? { status_reason: statusReason.trim() } : {};
      const updated = await client.rejectPlan(taskId, body);
      setTasks((prev) => mergeTaskIntoList(prev, updated));
      await refreshActiveBoard();
    },
    [client, refreshActiveBoard],
  );

  const saveTaskClarifications = useCallback(
    async (taskId: string, clarificationsJson: string) => {
      const updated = await client.patchTask(taskId, { clarifications_json: clarificationsJson });
      setTasks((prev) => mergeTaskIntoList(prev, updated));
    },
    [client],
  );

  const dispatchTaskById = useCallback(
    async (taskId: string, estimatedCost = 0) => {
      const updated = await client.dispatchTask(taskId, estimatedCost);
      setTasks((prev) => mergeTaskIntoList(prev, updated));
      await refreshActiveBoard();
    },
    [client, refreshActiveBoard],
  );

  const completeTaskById = useCallback(
    async (taskId: string) => {
      const updated = await client.completeTask(taskId);
      setTasks((prev) => mergeTaskIntoList(prev, updated));
      await refreshActiveBoard();
    },
    [client, refreshActiveBoard],
  );

  const stallNudgeTask = useCallback(
    async (taskId: string, note?: string) => {
      const updated = await client.stallNudge(taskId, note);
      setTasks((prev) => mergeTaskIntoList(prev, updated));
      await refreshActiveBoard();
    },
    [client, refreshActiveBoard],
  );

  const openTaskPullRequest = useCallback(
    async (taskId: string, headBranch: string, title?: string) => {
      await client.openPullRequest(taskId, {
        head_branch: headBranch.trim(),
        ...(title?.trim() ? { title: title.trim() } : {}),
      });
      const t = await client.getTask(taskId);
      setTasks((prev) => mergeTaskIntoList(prev, t));
    },
    [client],
  );

  const value = useMemo<MissionUiValue>(
    () => ({
      armsEnv: env,
      client,
      workspaces,
      activeWorkspace,
      productDetail,
      stalledTasks,
      tasks,
      agents,
      events: liveEvents,
      isOnline,
      feedLive,
      bumpFeedReconnect,
      boardLoading,
      boardLoadFailed,
      listLoading,
      apiError,
      goHome,
      openWorkspace,
      refreshWorkspaces,
      refreshActiveBoard,
      registerProduct,
      dismissError,
      fetchVersion,
      fetchOperationsLog,
      patchTaskStatus,
      createTaskForProduct,
      approveTaskPlan,
      rejectTaskPlan,
      saveTaskClarifications,
      dispatchTaskById,
      completeTaskById,
      stallNudgeTask,
      openTaskPullRequest,
    }),
    [
      env,
      client,
      workspaces,
      activeWorkspace,
      productDetail,
      stalledTasks,
      tasks,
      agents,
      liveEvents,
      isOnline,
      feedLive,
      bumpFeedReconnect,
      boardLoading,
      boardLoadFailed,
      listLoading,
      apiError,
      goHome,
      openWorkspace,
      refreshWorkspaces,
      refreshActiveBoard,
      registerProduct,
      dismissError,
      fetchVersion,
      fetchOperationsLog,
      patchTaskStatus,
      createTaskForProduct,
      approveTaskPlan,
      rejectTaskPlan,
      saveTaskClarifications,
      dispatchTaskById,
      completeTaskById,
      stallNudgeTask,
      openTaskPullRequest,
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
