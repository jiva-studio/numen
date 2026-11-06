import type { SlotCommand } from '@numen/plugin-sdk'

/** The words the bar writes itself. */
export const WORDS = {
  bar: 'Status bar',
} as const

/** The hover text of a command: its title and, where it has one, its shortcut. */
export function formatTooltip(command: SlotCommand): string {
  return command.shortcut ? `${command.title} (${command.shortcut})` : command.title
}
