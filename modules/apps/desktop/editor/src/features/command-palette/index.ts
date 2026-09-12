/**
 * The command palette: the window it opens, the commands it offers, and the
 * search that stands behind them.
 */
export { default as CommandPalette } from './CommandPalette.vue'
export { chorded, commandFor, keyOf, keysOf } from './chords'
export { commandsOf } from './list'
export { overNote } from './commands'
export { reaching } from './deps'
export type { CommandDeps, Notes, Store } from './deps'
export { lands } from './destination'
export type { DestinationDeps } from './destination'
export { does } from './handlers'
export { useCommandPalette } from './palette'
export type { Commands } from './palette'
export { runSupport } from './runs'
export type { RunSupport } from './runs'
export { useSearch } from './search'
export type { SearchDeps, SearchMode, SearchState } from './search'
export type { SearchDestination } from './lookup'
export { invocationOf } from './target'
export type { CommandsDeps, CommandTarget, VaultRef } from './target'
export type { NoteLookup, PaletteLists } from './lists'
