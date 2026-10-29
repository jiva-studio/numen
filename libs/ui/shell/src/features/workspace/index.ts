/** The panes a window is divided into, and the tabs standing in each of them. */
export { default as WorkspaceLayout } from './ui/WorkspaceLayout.vue'
export { closeTab, openTab, openTabAdjacent } from './lib/edit'
export { branch, pane } from './lib/node'
export type { Tab, Workspace } from './lib/node'
export { paneById, getPanes } from './lib/tree'
