/** The command: what a feature declares, what the palette lists and what carries it out. */
export type {
  Command,
  CommandGroup,
  CommandInvocation,
  CommandPlace,
  CommandPrompt,
  CommandTarget,
  ConfirmWords,
  FeatureCommands,
  PromptStep,
  RetypeWords,
  RunSupport,
  VaultRef,
} from './model/command'
export { ARGUMENT, COMMAND, COMMANDS_ROW, ROW, SETTING_COMMAND_PREFIX } from './lib/ids'
export { withWhen } from './model/availability'
export { createRowKeys, findChordKeys, findShortcutKeys } from './lib/keys'
export { findIcon } from './lib/icons'
export { flushTab } from './model/notes'
export type { Notes } from './model/notes'
export type { CommandDeps, CommandHandler } from './model/deps'
export { commands, registerCommand } from './model/registry'
