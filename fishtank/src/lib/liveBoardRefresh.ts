/**
 * arms `LiveActivityEvent.type` values that change task rows, PR fields, merge state, or convoy
 * enough that the Kanban should re-sync from REST (debounced in MissionUiContext).
 * Intentionally excludes chat and cost noise.
 */
const ARMS_SSE_TYPES_REFRESH_BOARD = new Set<string>([
  'task_dispatched',
  'task_completed',
  'task_stall_nudged',
  'task_execution_reassigned',
  'checkpoint_saved',
  'pull_request_opened',
  'merge_ship_completed',
  'convoy_subtask_dispatched',
  'convoy_subtask_completed',
]);

export function shouldRefreshBoardOnArmsSseType(armsType: string): boolean {
  return ARMS_SSE_TYPES_REFRESH_BOARD.has(armsType);
}
