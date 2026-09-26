import { taskProjectId } from './taskDisplay.js';

const byRelevance = (aCount, bCount, aId, bId, aTitle, bTitle) => {
  const diff = bCount - aCount;
  if (diff) return diff;
  return String(aTitle || '').localeCompare(String(bTitle || ''));
};

export const projectHue = (id) =>
  [...String(id || '')].reduce((hash, ch) => (hash * 33 + ch.charCodeAt(0)) % 360, 0);

export const projectColor = (id) => `hsl(${projectHue(id)} 62% 46%)`;

export const projectInitial = (title, id) =>
  String(title || id || '?')
    .trim()
    .slice(0, 1)
    .toUpperCase() || '?';

export const projectOpenCount = (tasks, projectId) =>
  (tasks ?? []).filter((task) => taskProjectId(task) === projectId && task.status !== 'archived')
    .length;

export const workspaceOpenCount = (tasks, workspaceId) =>
  (tasks ?? []).filter((task) => task.workspace_id === workspaceId && task.status !== 'archived')
    .length;

// Longest-prefix parent lookup: id "a/b/c" nests under "a/b" only if "a/b" is
// itself in the list, else falls back to "a", etc. Shared by the sidebar tree
// below and Project Settings' nested-project inspection, so both agree on
// what "child project" means at any nesting depth, not just one level.
export const parentIdOf = (ids, id) => {
  let parent = null;
  for (const candidate of ids) {
    if (candidate === id) continue;
    if (!id.startsWith(`${candidate}/`)) continue;
    if (!parent || candidate.length > parent.length) parent = candidate;
  }
  return parent;
};

export const childIdsOf = (ids, id) => {
  // `id` itself may not be a member of `ids` (e.g. a workspace id with no
  // matching project entry) — it still needs to count as a candidate
  // ancestor, otherwise nothing under it is ever recognized as a direct child.
  const candidates = ids.includes(id) ? ids : [...ids, id];
  return ids.filter((candidate) => candidate !== id && parentIdOf(candidates, candidate) === id);
};

const nestById = (projects, tasks) => {
  const nodes = projects.map((project) => ({
    project,
    children: [],
    taskCount: projectOpenCount(tasks, project.id)
  }));
  const byId = new Map(nodes.map((node) => [node.project.id, node]));
  const ids = nodes.map((node) => node.project.id);
  const roots = [];
  for (const node of nodes) {
    const parentId = parentIdOf(ids, node.project.id);
    const parent = parentId ? byId.get(parentId) : null;
    if (parent) parent.children.push(node);
    else roots.push(node);
  }
  const sortNodes = (list) => {
    list.sort((a, b) =>
      byRelevance(
        a.taskCount,
        b.taskCount,
        a.project.id,
        b.project.id,
        a.project.title || a.project.id,
        b.project.title || b.project.id
      )
    );
    for (const item of list) sortNodes(item.children);
  };
  sortNodes(roots);
  return roots;
};

export const buildSidebarTree = (workspaces, projects, tasks) =>
  [...(workspaces ?? [])]
    .sort((a, b) =>
      byRelevance(
        workspaceOpenCount(tasks, a.id),
        workspaceOpenCount(tasks, b.id),
        a.id,
        b.id,
        a.title,
        b.title
      )
    )
    .map((workspace) => {
      const owned = (projects ?? []).filter(
        (project) => (project.workspace_id || project.id) === workspace.id
      );
      const rootProject = owned.find((project) => project.id === workspace.id) || null;
      const childProjects = owned.filter((project) => project.id !== workspace.id);
      const children = nestById(childProjects, tasks);
      const isGroup = workspace.kind === 'workspace_group' || children.length > 0;
      const leafProject = isGroup
        ? null
        : rootProject ||
          owned[0] || { id: workspace.id, title: workspace.title, workspace_id: workspace.id };
      const rootNode =
        isGroup && rootProject
          ? {
              project: rootProject,
              children: [],
              taskCount: projectOpenCount(tasks, rootProject.id)
            }
          : null;
      return {
        workspace,
        isGroup,
        leafProject,
        rootNode,
        children,
        taskCount: workspaceOpenCount(tasks, workspace.id)
      };
    });
