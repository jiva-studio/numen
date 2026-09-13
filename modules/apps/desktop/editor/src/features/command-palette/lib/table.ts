/**
 * Construction of the full list of commands offered by the application.
 */
import { fileCommandsOf } from './fileCommands'
import { noteCommandsOf } from './noteCommands'
import { vaultCommandsOf } from './vaultCommands'
import { windowCommandsOf } from './windowCommands'
import type { Command } from '../types'
import type { Words } from '../words'

/**
 * Every command, in the order it is drawn. The keyboard it is being read on
 * decides how the keystrokes on it are written.
 *
 * A command reached by Shift and Enter on another one's row is offered here
 * too and drawn nowhere: the row it belongs to is the one that names it.
 */
export const commandsOf = (
  words: Words,
  agent: string = navigator.userAgent,
): readonly Command[] => [
  ...noteCommandsOf(words, agent),
  ...fileCommandsOf(words),
  ...windowCommandsOf(words, agent),
  ...vaultCommandsOf(words, agent),
]
