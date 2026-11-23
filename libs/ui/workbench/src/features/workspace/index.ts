/** The panes a window is divided into, and the tabs standing in each of them. */
export { default as WorkspaceLayout } from './ui/WorkspaceLayout.vue'
export { closeTab, openTab, openTabAdjacent } from './lib/edit'
export { branch, createPane } from './lib/node'
export type { Tab, Workspace } from './lib/node'
export type { TabMenu } from './ui/WorkspaceLayout.types'
export { findPane, getPanes } from './lib/tree'
export { getTabPaneRect } from './api/paneRect'
