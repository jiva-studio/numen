/**
 * Command grouping and filtering.
 */
import type { Command, CommandGroup } from '../../types'

export { createCommands } from './table'

/**
 * The commands offered on the row of another command. They are drawn only there.
 */
const getNestedCommandIds = (commands: readonly Command[]): ReadonlySet<string | undefined> =>
  new Set(commands.map((one) => one.also))

/**
 * The commands of one group, in the order they are drawn, less the ones
 * offered on another command's row.
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
 * Whether the typed text selects the commands: the field was empty, and
 * the typed character is the command prefix.
 */
export const isCommandPrefixTyped = (was: string, now: string): boolean => was === '' && now === '>'
