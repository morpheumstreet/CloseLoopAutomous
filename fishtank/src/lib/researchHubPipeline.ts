/**
 * Heuristic parsing of ResearchClaw `GET /api/pipeline/status` JSON.
 * Mirrors arms `researchclaw.pipelineLooksIdle` / running detection.
 */
export type PipelineRunState = 'running' | 'idle' | 'unknown';

export function inferPipelineRunState(json: unknown): PipelineRunState {
  if (json == null || typeof json !== 'object') return 'unknown';
  const o = json as Record<string, unknown>;
  if (o.pipeline_running === true || o.running === true || o.active === true) return 'running';
  if (o.pipeline_running === false || o.running === false || o.active === false) {
    if (typeof o.status === 'string') {
      const s = o.status.toLowerCase().trim();
      if (['running', 'in_progress', 'starting', 'active'].includes(s)) return 'running';
    }
    return 'idle';
  }
  if (typeof o.status === 'string') {
    const s = o.status.toLowerCase().trim();
    if (['idle', 'completed', 'stopped', 'done', 'ok'].includes(s)) return 'idle';
    if (['running', 'in_progress', 'starting', 'active'].includes(s)) return 'running';
  }
  return 'unknown';
}
