export interface BoardLocation {
  nav: string;
  project: string;
  workspace: string;
  view: string;
  filter: string;
  sort: string;
  q: string;
  section: string;
  sessions: string;
}

const defaults: BoardLocation = {
  nav: 'work',
  project: '',
  workspace: '',
  view: 'board',
  filter: 'all',
  sort: 'updated',
  q: '',
  section: 'general',
  sessions: 'active'
};

export function readBoardLocation(search: string): BoardLocation {
  const params = new URLSearchParams(search.startsWith('?') ? search.slice(1) : search);
  const pick = (key: keyof BoardLocation) => params.get(key) || defaults[key];
  return {
    nav: pick('nav'),
    project: pick('project'),
    workspace: pick('workspace'),
    view: pick('view'),
    filter: pick('filter'),
    sort: pick('sort'),
    q: pick('q'),
    section: pick('section'),
    sessions: pick('sessions')
  };
}

export function boardSearch(location: BoardLocation): string {
  const params = new URLSearchParams();
  (Object.keys(defaults) as (keyof BoardLocation)[]).forEach((key) => {
    const value = location[key] || '';
    if (value && value !== defaults[key]) params.set(key, value);
  });
  const text = params.toString();
  return text ? `?${text}` : '';
}

export function writeBoardLocation(location: BoardLocation) {
  const next = boardSearch(location);
  const current = window.location.search;
  if (next === current) return;
  window.history.replaceState(
    null,
    '',
    `${window.location.pathname}${next}${window.location.hash}`
  );
}

export interface KnownEntities {
  projectIds: Set<string>;
  workspaceIds: Set<string>;
  views: string[];
  viewNeedingProject: string;
}

/** Drops URL-supplied selection that data no longer backs (a stale bookmark
 * or a project that disappeared), so the board never sits on an unknown
 * project with its filter bar hidden. */
export function reconcileBoardLocation<
  T extends Pick<BoardLocation, 'project' | 'workspace' | 'view'>
>(location: T, known: KnownEntities): T {
  const project = known.projectIds.has(location.project) ? location.project : '';
  const workspace = known.workspaceIds.has(location.workspace) ? location.workspace : '';
  let view = known.views.includes(location.view) ? location.view : defaults.view;
  if (view === known.viewNeedingProject && !project && !workspace) view = defaults.view;
  return { ...location, project, workspace, view };
}
