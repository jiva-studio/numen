/**
 * Command grouping and filtering.
 */
import type { Command, CommandGroup } from '../types'

export { createCommands } from './table'

/**
 * The commands another one reaches on its own row. They are offered there and
 * drawn nowhere of their own.
 */
const getNestedCommandIds = (commands: readonly Command[]): ReadonlySet<string | undefined> =>
  new Set(commands.map((one) => one.also))

/**
 * The commands of one group, in the order they are drawn, less the ones
 * another command's row reaches.
 */
export const getGroupCommands = (
  commands: readonly Command[],
  group: CommandGroup,
): readonly Command[] => {
  const second = getNestedCommandIds(commands)
  return commands.filter((one) => one.group === group && !second.has(one.id))
}

/** The commands over the note in front, in the order they are drawn. */
export const getNoteCommands = (commands: readonly Command[]): readonly Command[] =>
  getGroupCommands(commands, 'note')

/**
 * Whether what was typed asks for the commands: the field held nothing, and
 * what went into it is the one character that means them.
 */
export const isCommandPrefixTyped = (was: string, now: string): boolean => was === '' && now === '>'
