import type { TitleBarCommand } from '@numen/plugin-sdk'

/** The ids of the commands that begin a group after the first one, where a separator is drawn. */
export function findGroupStarts(commands: readonly TitleBarCommand[]): ReadonlySet<string> {
  const starts = new Set<string>()
  commands.forEach((command, index) => {
    const before = commands[index - 1]
    if (before && before.group !== command.group) starts.add(command.id)
  })
  return starts
}
