/**
 * The command palette: the window it opens, the commands it offers, and the
 * search that stands behind them.
 */
export { default as CommandPalette } from './ui/CommandPalette.vue'
export { isChord, commandFor, keyOf, keysOf } from './lib/chords'
export { commandsOf } from './lib/table'
export { overNote } from './lib/commands'
export { createNotes } from './deps'
export type { CommandDeps, Notes, Store } from './deps'
export { openDestination } from './model/destination'
export type { DestinationDeps } from './model/destination'
export { runInvocation } from './model/handlers'
export { useCommandPalette } from './model/palette'
export type { Commands } from './model/palette'
export { runSupport } from './runs'
export type { RunSupport } from './runs'
export { useSearch } from './model/search'
export type { SearchDeps, SearchMode, SearchState } from './model/search'
export type { SearchDestination } from './model/lookup'
export { invocationOf } from './target'
export type { CommandsDeps, CommandTarget, VaultRef } from './target'
export type { NoteLookup, PaletteLists } from './rows'
