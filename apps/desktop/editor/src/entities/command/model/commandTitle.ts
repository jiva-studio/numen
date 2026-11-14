import type { Command } from './command'

/** Reads the title of a command by its id; empty for an id none of `commands` has. */
export const createCommandTitleReader =
  (commands: readonly Pick<Command, 'id' | 'text'>[]) =>
  (id: string): string =>
    commands.find((one) => one.id === id)?.text ?? ''
