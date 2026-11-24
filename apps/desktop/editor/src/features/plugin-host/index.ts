/** The plugin host of the window: what the plugins are served, the slots they place entries in, and the services they call. */
export { createPluginNotes } from './model/pluginNotes'
export type { PluginNotes } from './model/pluginNotes'
export { createPluginSupport } from './model/createPluginSupport'
export {
  createNoticeParts,
  createTabActionParts,
  createTabKindParts,
  createTaskServiceParts,
  createVaultServiceParts,
} from './model/createHostParts'
export type { HostParts } from './model/createHostParts'
export { createDialogStack } from './model/createDialogStack'
export type { DialogStack } from './model/createDialogStack'
export { default as PluginDialogs } from './ui/PluginDialogs.vue'
export { createWindowPluginStatus } from './model/pluginSettings'
export {
  createVaultAccess,
  createWindowService,
  createWorkspaceService,
} from './model/createPluginServices'
export { createPluginCatalog } from './model/createPluginCatalog'
export { createPluginReports } from './model/createPluginReports'
export { getAppCommandIds } from './model/window/getAppCommandIds'
export { getListedCommandIds, isCommandListed } from './model/window/listedCommands'
export { createVaultFlags } from './model/window/createVaultFlags'
export { createWindowFacts } from './model/window/createWindowFacts'
export { createWindowSlots } from './model/window/createWindowSlots'
export { createSlotGroups } from './model/window/slotGroups'
export { createNoteKindReader } from './model/window/createNoteKindReader'
export { createCommandIconFinder } from './model/window/createCommandIconFinder'
export { createHostPort } from './model/hostPort'
export type { HostPort } from './model/hostPort'
export type { PluginServices } from './model/pluginServices'
export { createWindowCommands } from './model/window/windowCommands'
export { createWindowVaults } from './model/window/createWindowVaults'
export { createSearchMenu } from './lib/searchMenu'
export { createTextSaver, type TextSaver } from './model/createTextSaver'
