/**
 * The commands the palette itself owns. The commands of the features come to the palette from the
 * window that collects them.
 */
import { createWindowCommands } from '../window-commands/windowCommands'
import type { Command } from '@/entities/command'
import type { Words } from '../../words'

/** The commands the palette owns, in the order they are drawn. `agent` is the user agent that picks the key chords. */
export const createCommands = (words: Words, agent: string): readonly Command[] =>
  createWindowCommands(words, agent)
