/** The panes a window is divided into, and the tabs standing in each of them. */
export { default as WorkspaceLayout } from './ui/WorkspaceLayout.vue'
export { WorkspacePane } from './ui/pane'
export { closeTab, openTab, openTabBeside } from './lib/edit'
export { branch, pane } from './lib/node'
export type { Tab, Workspace } from './lib/node'
export { paneById, panesOf } from './lib/tree'
