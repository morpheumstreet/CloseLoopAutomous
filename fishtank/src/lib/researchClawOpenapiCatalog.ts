/**
 * Catalog derived from ResearchClaw OpenAPI (info.version 0.3.3).
 * Keep in sync with the canonical spec (e.g. `videodecodetg/openapi.yaml` in your tooling tree) when routes change.
 * REST paths only — WebSockets are listed in RESEARCH_CLAW_WEBSOCKET_NOTES.
 */

export const RESEARCH_CLAW_OPENAPI_SPEC_VERSION = '0.3.3';
export const RESEARCH_CLAW_OPENAPI_SPEC_TITLE = 'ResearchClaw';

export const RESEARCH_CLAW_WEBSOCKET_NOTES = [
  'WebSocket /ws/events — real-time event stream for the dashboard',
  'WebSocket /ws/chat — conversational research chat (when not running with dashboard_only)',
] as const;

export type ResearchClawOpTag = 'core' | 'pipeline' | 'projects' | 'voice';

export type ResearchClawCatalogOp = {
  id: string;
  tag: ResearchClawOpTag;
  method: 'GET' | 'POST';
  /** Path with {param} placeholders, or final path when no params */
  path: string;
  summary: string;
  description?: string;
  /** When false, UI shows why (e.g. multipart) — not sent via arms invoke */
  invokeSupported: boolean;
  invokeNote?: string;
  pathParams?: { name: string; description?: string }[];
  /** Default body for POST (invokeSupported) */
  bodyJsonExample?: unknown;
};

export const RESEARCH_CLAW_OPERATIONS: ResearchClawCatalogOp[] = [
  {
    id: 'get_health',
    tag: 'core',
    method: 'GET',
    path: '/api/health',
    summary: 'Health',
    invokeSupported: true,
  },
  {
    id: 'get_version',
    tag: 'core',
    method: 'GET',
    path: '/api/version',
    summary: 'Version info',
    description: 'Package, HTTP API contract, runtime, and OpenAPI spec metadata.',
    invokeSupported: true,
  },
  {
    id: 'get_config',
    tag: 'core',
    method: 'GET',
    path: '/api/config',
    summary: 'Config summary',
    invokeSupported: true,
  },
  {
    id: 'post_pipeline_start',
    tag: 'pipeline',
    method: 'POST',
    path: '/api/pipeline/start',
    summary: 'Start pipeline',
    description: 'Start a new pipeline run.',
    invokeSupported: true,
    bodyJsonExample: { topic: null, config_overrides: null, auto_approve: true },
  },
  {
    id: 'post_pipeline_stop',
    tag: 'pipeline',
    method: 'POST',
    path: '/api/pipeline/stop',
    summary: 'Stop pipeline',
    description: 'Stop the currently running pipeline.',
    invokeSupported: true,
    bodyJsonExample: {},
  },
  {
    id: 'get_pipeline_status',
    tag: 'pipeline',
    method: 'GET',
    path: '/api/pipeline/status',
    summary: 'Pipeline status',
    description: 'Get current pipeline run status.',
    invokeSupported: true,
  },
  {
    id: 'get_pipeline_stages',
    tag: 'pipeline',
    method: 'GET',
    path: '/api/pipeline/stages',
    summary: 'Pipeline stages',
    description: 'Get the 23-stage pipeline definition.',
    invokeSupported: true,
  },
  {
    id: 'get_runs',
    tag: 'pipeline',
    method: 'GET',
    path: '/api/runs',
    summary: 'List runs',
    description: 'List historical pipeline runs from artifacts/ directory.',
    invokeSupported: true,
  },
  {
    id: 'get_run',
    tag: 'pipeline',
    method: 'GET',
    path: '/api/runs/{run_id}',
    summary: 'Get run',
    description: 'Get details for a specific run.',
    invokeSupported: true,
    pathParams: [{ name: 'run_id', description: 'Run id' }],
  },
  {
    id: 'get_run_metrics',
    tag: 'pipeline',
    method: 'GET',
    path: '/api/runs/{run_id}/metrics',
    summary: 'Get run metrics',
    description: 'Get experiment metrics for a run.',
    invokeSupported: true,
    pathParams: [{ name: 'run_id', description: 'Run id' }],
  },
  {
    id: 'get_projects',
    tag: 'projects',
    method: 'GET',
    path: '/api/projects',
    summary: 'List projects',
    description: 'List all project directories (artifacts/rc-*).',
    invokeSupported: true,
  },
  {
    id: 'post_voice_transcribe',
    tag: 'voice',
    method: 'POST',
    path: '/api/voice/transcribe',
    summary: 'Transcribe audio',
    description: 'Multipart upload; registered only when server.voice_enabled is true.',
    invokeSupported: false,
    invokeNote: 'Multipart form upload — use ResearchClaw directly or curl; not proxied here.',
  },
];

const TAG_LABEL: Record<ResearchClawOpTag, string> = {
  core: 'Core',
  pipeline: 'Pipeline',
  projects: 'Projects',
  voice: 'Voice',
};

export function researchClawTagLabel(tag: ResearchClawOpTag): string {
  return TAG_LABEL[tag];
}

export function groupResearchClawOpsByTag(): Record<ResearchClawOpTag, ResearchClawCatalogOp[]> {
  const out: Record<ResearchClawOpTag, ResearchClawCatalogOp[]> = {
    core: [],
    pipeline: [],
    projects: [],
    voice: [],
  };
  for (const op of RESEARCH_CLAW_OPERATIONS) {
    out[op.tag].push(op);
  }
  return out;
}

/** Replace `{run_id}` with encoded segment values. */
export function buildResearchClawPath(op: ResearchClawCatalogOp, pathValues: Record<string, string>): string {
  let p = op.path;
  for (const [k, raw] of Object.entries(pathValues)) {
    const v = (raw ?? '').trim();
    p = p.split(`{${k}}`).join(encodeURIComponent(v));
  }
  if (p.includes('{')) {
    throw new Error('Missing path parameter');
  }
  return p;
}
