import { useOutletContext } from 'react-router-dom';
import { MissionQueuePanel } from '../components/workspace/MissionQueuePanel';
import type { WorkspaceMainOutletContext } from './workspaceMainOutletContext';

export function MissionTasksPage() {
  const ctx = useOutletContext<WorkspaceMainOutletContext>();
  return (
    <MissionQueuePanel
      boardSearch={ctx.boardSearch}
      assigneeAgentId={ctx.assigneeAgentId}
      onAssigneeAgentIdChange={ctx.onAssigneeAgentIdChange}
      newTaskOpen={ctx.newTaskOpen}
      onNewTaskOpenChange={ctx.onNewTaskOpenChange}
    />
  );
}
