/**
 * The command palette: the window it opens, the commands it offers, and the
 * search that stands behind them.
 */
export { default as CommandPalette } from './ui/CommandPalette.vue'
export { isChord, commandFor, keyOf, keysOf } from './lib/chords'
export { commandsOf } from './lib/table'
export { overNote } from './lib/commands'
export { invocationOf } from './lib/invocation'
export { createNotes } from './lib/notes'
export { openDestination } from './model/destination'
export type { DestinationDeps } from './model/destination'
export { runInvocation } from './model/handlers'
export { useCommandPalette } from './model/palette'
export type { Commands } from './model/palette'
export { runSupport } from './model/runs'
export { useSearch } from './model/search'
export type { SearchDeps, SearchMode, SearchState } from './model/search'
export type { SearchDestination } from './model/lookup'
export type { CommandDeps } from './model/deps'
export { ANSWER_WORDS } from './words'
export type { AnswerWords } from './words'
export type {
  CommandsDeps,
  CommandTarget,
  NoteLookup,
  Notes,
  PaletteLists,
  RunSupport,
  Store,
  VaultRef,
} from './types'
