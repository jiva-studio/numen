/**
 * The engine of the palette: the ids of the commands, what carries one out, the steps it prompts
 * in, the keys and icons it is drawn with, and the files it reaches.
 */
export { ARGUMENT, COMMAND, COMMANDS_ROW, ROW, SETTING_COMMAND_PREFIX } from './ids'
export type {
  CommandInvocation,
  CommandTarget,
  InvocationTarget,
  RunSupport,
  VaultRef,
} from './invocation'
export { withWhen } from './availability'
export { getPaletteValues } from './paletteValues'
export type { StepGroup, StepRow } from './steps'
export { getValues, withValues, type Values } from './values'
export { createRowKeys, findShortcutKeys } from './keys'
export { findIcon } from './icons'
export { flushTab } from './notes'
export type { CommandPlace, Notes } from './notes'
export type { CommandDeps, CommandHandler } from './deps'
export { createCommandRegistry } from './registry'
export type { CommandRegistry } from './registry'
