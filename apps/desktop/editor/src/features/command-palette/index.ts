/**
 * The command palette: the window it opens, the commands it offers, and the
 */
export { default as CommandPalette } from './ui/CommandPalette.vue'
export {
  isChord,
  findChordCommand,
  findChordKeys,
  createRowKeys,
  findShortcutKeys,
} from './lib/chords/chords'
export { createCommands } from './lib/commands/table'
export { createPathCommands } from './lib/commands/pathMenu'
export type { MenuRow, PathContribution, PathItem } from './lib/commands/pathMenu'
export { getNoteCommands } from './lib/commands/commands'
export { createInvocation } from './lib/offers/invocation'
export { createNotes } from './lib/notes/notes'
export { openDestination } from './model/search/destination'
export type { DestinationDeps } from './model/search/destination'
export { commands, registerCommand, runInvocation } from './model/handlers/handlers'
export { useCommandPalette } from './model/palette/palette'
export type { Commands } from './model/palette/palette'
export { runSupport } from './model/palette/runSupport'
export { useSearch } from './model/search/search'
export type { SearchDeps, SearchMode, SearchState } from './model/search/search'
export type { SearchDestination } from './model/search/lookup'
export type { CommandDeps, VaultWriter } from './model/handlers/deps'
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
