import { PROJECT_SETTINGS_VIEW } from './projectSettings.ts';
import { reconcileBoardLocation, type BoardLocation } from './urlState.ts';

/**
 * How each sidebar/toolbar action moves the board location. Every function is
 * pure: it returns the next location and never touches the current one.
 */

const BOARD_VIEWS = ['board', 'archive', PROJECT_SETTINGS_VIEW];

/** A location read from the URL, with values this build does not know folded to defaults. */
export const normalizeLocation = (location: BoardLocation): BoardLocation => ({
  ...location,
  nav: location.nav === 'settings' ? 'settings' : 'work',
  sessions: location.sessions === 'closed' ? 'closed' : 'active'
});

export const selectProject = (location: BoardLocation, projectId: string): BoardLocation => ({
  ...location,
  project: projectId,
  workspace: '',
  nav: 'work',
  view: 'board'
});

// A workspace-level selection has no single project's Settings tab to stay
// on, so leaving it pinned there would show a project page for no project.
const leaveProjectSettings = (view: string): string =>
  view === PROJECT_SETTINGS_VIEW ? 'board' : view;

export const selectWorkspace = (location: BoardLocation, workspaceId: string): BoardLocation => ({
  ...location,
  workspace: workspaceId,
  project: '',
  nav: 'work',
  view: leaveProjectSettings(location.view)
});

export const selectAllProjects = (location: BoardLocation): BoardLocation => ({
  ...location,
  project: '',
  workspace: '',
  view: leaveProjectSettings(location.view)
});

export const selectQuickView = (location: BoardLocation, viewId: string): BoardLocation => ({
  ...location,
  project: '',
  workspace: '',
  nav: 'work',
  ...(viewId === 'archived'
    ? { view: 'archive', filter: 'all' }
    : { view: 'board', filter: viewId })
});

export const changeChip = (location: BoardLocation, chipId: string): BoardLocation => ({
  ...location,
  filter: chipId,
  view: 'board'
});

export const selectNav = (location: BoardLocation, nav: string): BoardLocation => {
  if (nav === 'agents') return { ...location, section: 'agents', nav: 'settings' };
  if (nav === 'settings') return { ...location, nav: 'settings' };
  const view = nav === 'work' && !BOARD_VIEWS.includes(location.view) ? 'board' : location.view;
  return { ...location, nav, view };
};

export const selectSettingsSection = (location: BoardLocation, section: string): BoardLocation => ({
  ...location,
  section,
  nav: 'settings'
});

/** Drops selection that the loaded data no longer backs. */
export const reconcileSelection = (
  location: BoardLocation,
  projectIds: string[],
  workspaceIds: string[]
): BoardLocation =>
  reconcileBoardLocation(location, {
    projectIds: new Set(projectIds),
    workspaceIds: new Set(workspaceIds),
    views: BOARD_VIEWS,
    viewNeedingProject: PROJECT_SETTINGS_VIEW
  });
