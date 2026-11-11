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
  InvocationTarget,
  PromptStep,
  RetypeWords,
  RunSupport,
  VaultRef,
} from './model/command'
export { ARGUMENT, COMMAND, COMMANDS_ROW, ROW, SETTING_COMMAND_PREFIX } from './lib/ids'
export { withWhen } from './model/availability'
export { getPaletteValues } from './model/paletteValues'
export type { StepGroup, StepRow } from './model/steps'
export { getValues, withValues, type Values } from './model/values'
export { createRowKeys, findShortcutKeys } from './lib/keys'
export { findIcon } from './lib/icons'
export { flushTab } from './model/notes'
export type { Notes } from './model/notes'
export type { CommandDeps, CommandHandler } from './model/deps'
export { createCommandRegistry } from './model/registry'
export type { CommandRegistry } from './model/registry'
