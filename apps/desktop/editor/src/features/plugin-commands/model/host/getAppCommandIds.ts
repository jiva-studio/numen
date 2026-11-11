import type { Command } from '@/entities/command'

/** The ids of the commands of the window, once each: the command a row stands for, else the row. */
export const getAppCommandIds = (commands: readonly Command[]): string[] => [
  ...new Set(commands.map((one) => one.command ?? one.id)),
]
