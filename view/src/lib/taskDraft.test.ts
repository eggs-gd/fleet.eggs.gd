import assert from 'node:assert/strict';
import test from 'node:test';
import {
  draftFromTask,
  draftIsDirty,
  draftNeedsReviewComment,
  emptyTaskDraft,
  taskPatchFromDraft
} from './taskDraft.ts';
import type { Project, Task, Workspace } from './types.ts';

const task: Task = {
  ref: 'T-1',
  status: 'todo',
  priority: 3,
  assignee: 'pavlo',
  project_id: 'api',
  milestone: 'm1',
  body: 'Body',
  depends_on: ['T-0']
};

const projects: Project[] = [
  { id: 'api', title: 'API', workspace_id: 'acme' },
  { id: 'web', title: 'Web', workspace_id: 'acme', repositories: ['web-repo'] }
];
const workspaces: Workspace[] = [{ id: 'acme', title: 'Acme', repositories: ['acme-repo'] }];

test('a draft starts as a copy of its task and is not dirty', () => {
  const draft = draftFromTask(task);
  assert.deepEqual(draft, {
    status: 'todo',
    priority: 3,
    assignee: 'pavlo',
    project: 'api',
    milestone: 'm1',
    dependsOn: 'T-0',
    body: 'Body',
    comment: ''
  });
  assert.equal(draftIsDirty(task, draft), false);
  assert.equal(emptyTaskDraft().priority, 5);
});

test('any edit, or a typed comment, makes the draft dirty', () => {
  const edits: Array<(d: ReturnType<typeof draftFromTask>) => void> = [
    (d) => (d.status = 'doing'),
    (d) => (d.priority = 1),
    (d) => (d.assignee = ''),
    (d) => (d.project = 'web'),
    (d) => (d.dependsOn = ''),
    (d) => (d.milestone = 'm2'),
    (d) => (d.body = 'changed'),
    (d) => (d.comment = 'note')
  ];
  for (const edit of edits) {
    const draft = draftFromTask(task);
    edit(draft);
    assert.equal(draftIsDirty(task, draft), true);
  }
  const blankComment = draftFromTask(task);
  blankComment.comment = '   ';
  assert.equal(draftIsDirty(task, blankComment), false);
});

test('a milestone choice is ignored once the project changes', () => {
  const draft = draftFromTask(task);
  draft.project = 'web';
  draft.milestone = 'm1';
  assert.equal(taskPatchFromDraft(task, draft, projects, workspaces, 'me').milestone, undefined);
});

test('the patch carries the editable fields and trims text', () => {
  const draft = draftFromTask(task);
  draft.status = 'doing';
  draft.assignee = ' codex ';
  draft.comment = ' go ';
  draft.dependsOn = 'T-0, T-5';
  const patch = taskPatchFromDraft(task, draft, projects, workspaces, 'me');
  assert.equal(patch.status, 'doing');
  assert.equal(patch.assignee, 'codex');
  assert.equal(patch.comment, 'go');
  assert.equal(patch.comment_author, 'me');
  assert.deepEqual(patch.depends_on, ['T-0', 'T-5']);
  assert.equal(patch.project, undefined);
  assert.equal(patch.milestone, undefined);
});

test('changing milestone alone sends it, including clearing it', () => {
  const draft = draftFromTask(task);
  draft.milestone = '';
  assert.equal(taskPatchFromDraft(task, draft, projects, workspaces, 'me').milestone, '');
});

test('moving project picks the first repository of the target, else the workspace', () => {
  const toWeb = draftFromTask(task);
  toWeb.project = 'web';
  const web = taskPatchFromDraft(task, toWeb, projects, workspaces, 'me');
  assert.equal(web.project, 'web');
  assert.equal(web.repository, 'web-repo');

  const toWorkspace = draftFromTask(task);
  toWorkspace.project = 'acme';
  const ws = taskPatchFromDraft(task, toWorkspace, projects, workspaces, 'me');
  assert.equal(ws.repository, 'acme-repo');
});

test('blocked to needs_review requires a comment', () => {
  const blocked: Task = { ...task, status: 'blocked' };
  const draft = draftFromTask(blocked);
  draft.status = 'needs_review';
  assert.equal(draftNeedsReviewComment(blocked, draft), true);
  draft.comment = 'fixed upstream';
  assert.equal(draftNeedsReviewComment(blocked, draft), false);
  assert.equal(draftNeedsReviewComment(task, { ...draft, comment: '' }), false);
});
