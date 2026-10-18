import { dependsOnDraft, parseDependsOnDraft, priorityValue } from './taskDisplay.ts';
import type { AnyRecord, Project, Task, Workspace } from './types.ts';

/** The editable fields of the task modal, held as the operator types. */
export interface TaskDraft {
  status: string;
  priority: number;
  assignee: string;
  project: string;
  milestone: string;
  dependsOn: string;
  body: string;
  comment: string;
}

export const emptyTaskDraft = (): TaskDraft => ({
  status: '',
  priority: 5,
  assignee: '',
  project: '',
  milestone: '',
  dependsOn: '',
  body: '',
  comment: ''
});

export const taskProjectOf = (task: Task): string => task.project_id || task.project || '';

export const draftFromTask = (task: Task): TaskDraft => ({
  status: task.status || '',
  priority: priorityValue(task),
  assignee: task.assignee || '',
  project: taskProjectOf(task),
  milestone: task.milestone || '',
  dependsOn: dependsOnDraft(task),
  body: task.body || '',
  comment: ''
});

/** Whether closing the modal would throw away something the operator typed. */
export const draftIsDirty = (task: Task, draft: TaskDraft): boolean => {
  const projectChanged = draft.project !== taskProjectOf(task);
  return (
    draft.status !== (task.status || '') ||
    Number(draft.priority) !== priorityValue(task) ||
    draft.assignee !== (task.assignee || '') ||
    projectChanged ||
    draft.dependsOn !== dependsOnDraft(task) ||
    (!projectChanged && draft.milestone !== (task.milestone || '')) ||
    draft.body !== (task.body || '') ||
    draft.comment.trim() !== ''
  );
};

/** blocked -> needs_review is operator recovery; it needs a review reason. */
export const BLOCKED_REVIEW_COMMENT_ERROR =
  'Add a review comment explaining why this blocked task is ready for needs_review.';

export const draftNeedsReviewComment = (task: Task, draft: TaskDraft): boolean =>
  task.status === 'blocked' && draft.status === 'needs_review' && !draft.comment.trim();

/** The PATCH body that turns `task` into what `draft` describes. */
export const taskPatchFromDraft = (
  task: Task,
  draft: TaskDraft,
  projects: Project[],
  workspaces: Workspace[],
  commentAuthor: string
): AnyRecord => {
  const patch: AnyRecord = {
    status: draft.status,
    priority: Number(draft.priority),
    assignee: draft.assignee.trim(),
    depends_on: parseDependsOnDraft(draft.dependsOn),
    body: draft.body,
    comment: draft.comment.trim(),
    comment_author: commentAuthor
  };
  const nextProject = draft.project.trim();
  if (nextProject && nextProject !== taskProjectOf(task)) {
    const project = projects.find((item) => item.id === nextProject);
    const repositoryOptions = project?.repositories?.length
      ? project.repositories
      : workspaces.find((workspace) => workspace.id === (project?.workspace_id || nextProject))
          ?.repositories || [];
    patch.project = nextProject;
    patch.repository = repositoryOptions[0] || '';
  } else if (draft.milestone !== (task.milestone || '')) {
    // A project move drops the milestone server-side, so only send it
    // when the project is unchanged.
    patch.milestone = draft.milestone;
  }
  return patch;
};
