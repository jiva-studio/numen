/**
 * The command palette: the window it opens, the commands it offers, and the searches it asks of
 * the core.
 */
export { default as CommandPalette } from './ui/CommandPalette.vue'
export { createCommands } from './lib/commands/table'
export { createInvocation } from './lib/entries/invocation'
export { createNotes } from './lib/notes/notes'
export type { FileIcons } from './model/rowIcons'
export { commands, registerCommand, runInvocation } from './model/handlers/handlers'
export { createCommandPalette } from './model/palette/palette'
export type { Commands } from './model/palette/palette'
export { runSupport } from './model/palette/runSupport'
export { searchCore } from './api/search'
export { useSearch } from './model/search/search'
export type { SearchDeps, SearchMode, SearchState } from './model/search/search'
export { createNameItem, resolveChoice } from './model/search/lookup'
export type { NameMatch, ResultChoice, SearchMenuDeps } from './model/search/lookup'
export type { CommandDeps } from './model/handlers/deps'
export { PALETTE_WORDS } from './words'
export type { Words } from './words'
export type { CommandsDeps, NoteLookup, PaletteLists, Store } from './types'
