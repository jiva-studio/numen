/**
 * The command palette: the window it opens, the commands it offers, and the
 * search that stands behind them.
 */
export { default as CommandPalette } from './ui/CommandPalette.vue'
export { isChord, findChordCommand, findChordKeys, createRowKeys } from './lib/chords'
export { createCommands } from './lib/table'
export { createPathCommands } from './lib/pathMenu'
export type { MenuRow, PathItem, RunGuard } from './lib/pathMenu'
export { getNoteCommands } from './lib/commands'
export { createInvocation } from './lib/invocation'
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
  Command,
  CommandsDeps,
  CommandTarget,
  NoteLookup,
  Notes,
  PaletteLists,
  RunSupport,
  Store,
  VaultRef,
} from './types'
