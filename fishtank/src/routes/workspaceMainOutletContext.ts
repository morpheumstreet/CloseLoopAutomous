/** Passed from `WorkspaceShellLayout` to task-board child routes via `<Outlet context>`. */
export type WorkspaceMainOutletContext = {
  boardSearch: string;
  onBoardSearchChange: (v: string) => void;
  assigneeAgentId: string | null;
  onAssigneeAgentIdChange: (id: string | null) => void;
  newTaskOpen: boolean;
  onNewTaskOpenChange: (open: boolean) => void;
};
