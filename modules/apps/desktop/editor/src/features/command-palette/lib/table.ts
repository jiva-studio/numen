/**
 * Construction of the full list of commands offered by the application.
 */
import { createArtifactCommands, createDeletionCommands } from './fileCommands'
import { createNoteCommands } from './noteCommands'
import { createVaultCommands } from './vaultCommands'
import {
  createWindowCreationCommands,
  createWindowDisplayCommands,
  createWindowFeatureCommands,
  createWindowNavigationCommands,
} from './windowCommands'
import type { Command } from '../types'
import type { Words } from '../words'

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
  ...createArtifactCommands(words),
  ...createDeletionCommands(words),
  ...createWindowCreationCommands(words, agent),
  ...createWindowNavigationCommands(words, agent),
  ...createWindowDisplayCommands(words),
  ...createWindowFeatureCommands(words),
  ...createVaultCommands(words, agent),
]
