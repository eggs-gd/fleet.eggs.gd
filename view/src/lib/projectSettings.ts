import { childIdsOf, parentIdOf } from './projectTree.ts';
import { repoWebUrl, techRepositories } from './technologyDisplay.ts';
import type { Project, Workspace, Repository } from './types.ts';

export const PROJECT_SETTINGS_VIEW = 'projectSettings';

// Longest-prefix match against real project/workspace ids, not a plain
// string split — so a project nested two or more levels deep (e.g.
// "audiophile/jivemax/jivelite") resolves the same parent the sidebar tree
// shows, instead of a one-level-only approximation. `candidateIds` should
// include both project and workspace ids, since a project's nearest ancestor
// may be a workspace with no matching project entry.
export const parentProjectId = (id: string, candidateIds: string[] = []): string =>
  parentIdOf(candidateIds, id) || '';

export const childProjects = (projects: Project[] | null | undefined, id: string): Project[] => {
  const ids = (projects || []).map((project) => project.id);
  const childIds = new Set(childIdsOf(ids, id));
  return (projects || []).filter((project) => childIds.has(project.id));
};

interface RepositoryIndex {
  byId: Map<string, Repository>;
  byPath: Map<string, Repository>;
}

const indexRepositories = (repositories: Repository[] | null | undefined): RepositoryIndex => {
  const byId = new Map<string, Repository>();
  const byPath = new Map<string, Repository>();
  for (const repo of repositories || []) {
    if (repo?.id) byId.set(repo.id, repo);
    if (repo?.relative_path) byPath.set(repo.relative_path, repo);
  }
  return { byId, byPath };
};

const lookupRepo = (
  repo: Repository | null | undefined,
  index: RepositoryIndex
): Repository | null => {
  if (repo?.id && index.byId.has(repo.id)) return index.byId.get(repo.id) || null;
  if (repo?.relative_path && index.byPath.has(repo.relative_path))
    return index.byPath.get(repo.relative_path) || null;
  return null;
};

const nestedLabels = (ids: string[] | null | undefined, index: RepositoryIndex): string[] =>
  (ids || []).map((nestedId) => {
    const nested = index.byId.get(nestedId);
    return nested?.relative_path || nested?.name || nestedId;
  });

export interface RepositoryInspectRow {
  id: string;
  name: string;
  relativePath: string;
  branch: string;
  remote: string;
  webUrl: string;
  nestedUnder: string;
  nests: string[];
}

export const repositoryInspectRow = (
  repo: Repository | null | undefined,
  repositories: Repository[] | RepositoryIndex = []
): RepositoryInspectRow => {
  const index = Array.isArray(repositories) ? indexRepositories(repositories) : repositories;
  const extra = lookupRepo(repo, index) || {};
  const remote = repo?.remote || extra.remote || '';
  const parentId = repo?.parent_repository_id || extra.parent_repository_id || '';
  const parent = parentId ? index.byId.get(parentId) : null;
  const nestedIds = repo?.nested_repository_ids || extra.nested_repository_ids || [];
  return {
    id: repo?.id || extra.id || '',
    name: repo?.name || extra.name || '',
    relativePath: repo?.relative_path || extra.relative_path || '',
    branch: repo?.branch || extra.branch || '',
    remote,
    webUrl: repoWebUrl(remote),
    nestedUnder: parent?.relative_path || parent?.name || parentId,
    nests: nestedLabels(nestedIds, index)
  };
};

export interface ProjectSettingsInspect {
  tag: string;
  kind: string;
  source: string;
  status: string;
  reviewStatus: string;
  projectMd: string;
  summary: string;
  membership: { id: string; title: string } | null;
  parent: { id: string; title: string } | null;
  nested: { id: string; title: string }[];
  repositories: RepositoryInspectRow[];
}

export const projectSettingsInspect = (
  item: Project | null | undefined,
  {
    workspaces = [],
    projects = [],
    repositories = []
  }: { workspaces?: Workspace[]; projects?: Project[]; repositories?: Repository[] } = {}
): ProjectSettingsInspect | null => {
  if (!item) return null;
  const index = indexRepositories(repositories);
  const workspace = workspaces.find((entry) => entry.id === (item.workspace_id || item.id)) || null;
  const candidateIds = [
    ...projects.map((entry) => entry.id),
    ...workspaces.map((entry) => entry.id)
  ];
  const parentId = parentProjectId(item.id, candidateIds);
  const parent =
    (parentId && projects.find((entry) => entry.id === parentId)) ||
    (parentId && workspaces.find((entry) => entry.id === parentId)) ||
    null;

  const seen = new Set<string>();
  const rows: RepositoryInspectRow[] = [];
  for (const repo of techRepositories(item)) {
    const key = repo.relative_path || repo.id;
    if (key) seen.add(key);
    rows.push(repositoryInspectRow(repo, index));
  }
  for (const name of item.repositories || []) {
    if (seen.has(name)) continue;
    seen.add(name);
    rows.push(repositoryInspectRow(index.byPath.get(name) || { relative_path: name }, index));
  }

  return {
    tag: workspace?.tag || item.tag || '',
    kind: item.kind || '',
    source: item.source || '',
    status: item.status || '',
    reviewStatus: item.review_status || '',
    projectMd: item.path || item.relative_path || '',
    summary: item.summary || '',
    membership:
      workspace && workspace.id !== item.id
        ? { id: workspace.id, title: workspace.title || workspace.id }
        : null,
    parent: parent
      ? { id: parent.id, title: parent.title || parent.id }
      : parentId
        ? { id: parentId, title: parentId }
        : null,
    nested: childProjects(projects, item.id).map((entry) => ({
      id: entry.id,
      title: entry.title || entry.id
    })),
    repositories: rows
  };
};
