/**
 * Construction of the full list of commands offered by the application.
 */
import { createNoteCommands } from '../note-commands/noteCommands'
import { createVaultCommands } from '../vault-commands/vaultCommands'
import {
  createWindowCreationCommands,
  createWindowNavigationCommands,
  createWindowSettingsCommands,
} from '../window-commands/windowCommands'
import type { Command } from '@/entities/command'
import type { Words } from '../../words'

/**
 * Every command, in the order it is drawn. The keyboard it is being read on
 * decides how the keystrokes on it are written.
 *
 * A command reached by Shift and Enter on another one's row is offered here
 * too and drawn nowhere: the row it belongs to is the one that names it.
 */
export const createCommands = (
  words: Words,
  agent: string = navigator.userAgent,
): readonly Command[] => [
  ...createNoteCommands(words, agent),
  ...createWindowCreationCommands(words, agent),
  ...createWindowNavigationCommands(words, agent),
  ...createWindowSettingsCommands(words),
  ...createVaultCommands(words, agent),
]
