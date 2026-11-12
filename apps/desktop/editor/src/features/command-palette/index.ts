/**
 * The command palette: the window it opens, the commands it offers, and the searches it asks of
 * the core.
 */
export { default as CommandPalette } from './ui/CommandPalette.vue'
export { createCommands } from './lib/commands/table'
export { createInvocation } from './lib/entries/invocation'
export { createNotes } from './lib/notes/notes'
export { runInvocation } from './model/handlers/handlers'
export type { Commands } from './model/palette/palette'
export { createCommandDeps } from './model/run/createCommandDeps'
export { createCommandNotes } from './model/notes/createCommandNotes'
export { createNoteLookup } from './model/notes/createNoteLookup'
export { createAppItemRunner } from './model/run/createAppItemRunner'
export { createSearchRequests } from './model/run/createSearchRequests'
export { createInvocationRunner } from './model/run/createInvocationRunner'
export { createVaultOpener } from './model/run/createVaultOpener'
export { runSupport } from './model/palette/runSupport'
export { searchCore } from './api/search'
export { PALETTE_WORDS } from './words'
export type { NoteLookup, PaletteLists } from './types'
export { createWindowPalette } from './model/window/createWindowPalette'
export type { SearchState } from './model/search/search'
export type { PluginPaletteLists } from './model/window/createWindowPalette'
