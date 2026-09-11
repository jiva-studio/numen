/**
 * Command grouping and filtering.
 */
import type { Command, CommandGroup } from './target'

export { commandsOf } from './list'

/**
 * The commands another one reaches on its own row. They are offered there and
 * drawn nowhere of their own.
 */
const secondary = (commands: readonly Command[]): ReadonlySet<string> =>
  new Set(commands.map((one) => one.also).filter((id) => id !== undefined))

/**
 * The commands of one group, in the order they are drawn, less the ones
 * another command's row reaches.
 */
export const inGroup = (
  commands: readonly Command[],
  group: CommandGroup,
): readonly Command[] => {
  const second = secondary(commands)
  return commands.filter((one) => one.group === group && !second.has(one.id))
}

/** The commands over the note in front, in the order they are drawn. */
export const overNote = (commands: readonly Command[]): readonly Command[] =>
  inGroup(commands, 'note')
