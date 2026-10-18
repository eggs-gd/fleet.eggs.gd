import { derived, writable, type Readable, type Writable } from 'svelte/store';
import {
  changeChip,
  normalizeLocation,
  reconcileSelection,
  selectAllProjects as allProjects,
  selectNav as nav,
  selectProject as project,
  selectQuickView as quickView,
  selectSettingsSection as settingsSection,
  selectWorkspace as workspace
} from '../lib/navigation';
import { readBoardLocation, writeBoardLocation, type BoardLocation } from '../lib/urlState';
import { data, projects, workspaces } from './board';

/** What the operator is looking at; mirrored into the URL so a reload or a shared link lands in the same place. */
export const boardLocation = writable<BoardLocation>(readBoardLocation(''));

const field = <K extends keyof BoardLocation>(key: K): Readable<BoardLocation[K]> =>
  derived(boardLocation, ($location) => $location[key]);

export const globalNav = field('nav');
export const selectedProjectId = field('project');
export const selectedWorkspaceId = field('workspace');
export const activeView = field('view');
export const quickFilter = field('filter');
export const taskSort = field('sort');
export const settingsSectionId = field('section');
export const sessionsTab = field('sessions');

/** The search box: readable and writable, so a component can `bind:` to it. */
export const query: Writable<string> = {
  subscribe: field('q').subscribe,
  set: (value) => boardLocation.update((current) => ({ ...current, q: value })),
  update: (fn) => boardLocation.update((current) => ({ ...current, q: fn(current.q) }))
};

const move = (step: (current: BoardLocation) => BoardLocation) => boardLocation.update(step);

export const selectProject = (id: string) => move((current) => project(current, id));
export const selectWorkspace = (id: string) => move((current) => workspace(current, id));
export const selectAllProjects = () => move(allProjects);
export const selectQuickView = (id: string) => move((current) => quickView(current, id));
export const changeFilter = (id: string) => move((current) => changeChip(current, id));
export const selectNav = (id: string) => move((current) => nav(current, id));
export const selectSettingsSection = (id: string) =>
  move((current) => settingsSection(current, id));
export const changeView = (view: string) => move((current) => ({ ...current, view }));
export const setSort = (sort: string) => move((current) => ({ ...current, sort }));
export const setSessionsTab = (tab: string) => move((current) => ({ ...current, sessions: tab }));

let known: { projectIds: string[]; workspaceIds: string[] } | null = null;

/** Takes the initial location from the URL, then keeps the URL and the loaded data in step with it. */
export function startLocationSync(search: string): () => void {
  boardLocation.set(normalizeLocation(readBoardLocation(search)));
  const stopUrl = boardLocation.subscribe(writeBoardLocation);
  // Selection the loaded data no longer backs (a stale bookmark, a removed
  // project) is dropped so the board never sits on an unknown project.
  const stopData = derived([data, projects, workspaces], ([$data, $projects, $workspaces]) =>
    $data
      ? {
          projectIds: $projects.map((item) => item.id),
          workspaceIds: $workspaces.map((item) => item.id)
        }
      : null
  ).subscribe((entities) => {
    known = entities;
    reconcile();
  });
  return () => {
    stopUrl();
    stopData();
    known = null;
  };
}

function reconcile() {
  if (!known) return;
  const { projectIds, workspaceIds } = known;
  boardLocation.update((current) => {
    const next = reconcileSelection(current, projectIds, workspaceIds);
    const same =
      next.project === current.project &&
      next.workspace === current.workspace &&
      next.view === current.view;
    return same ? current : next;
  });
}
