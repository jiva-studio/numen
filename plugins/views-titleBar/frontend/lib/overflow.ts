import type { TitleBarCommand } from '@numen/plugin-sdk'

export interface SplitCommands {
  readonly visible: readonly TitleBarCommand[]
  readonly overflow: readonly TitleBarCommand[]
}

/**
 * The commands that stay in the bar and the ones that go to the menu, in the order given. When all
 * of them fit they stay; otherwise the menu button takes one of the slots.
 */
export function splitCommands(
  commands: readonly TitleBarCommand[],
  capacity: number,
): SplitCommands {
  if (commands.length <= capacity) return { visible: commands, overflow: [] }
  const slots = Math.max(Math.floor(capacity) - 1, 0)
  return { visible: commands.slice(0, slots), overflow: commands.slice(slots) }
}
