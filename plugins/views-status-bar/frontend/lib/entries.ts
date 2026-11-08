import type { SlotCommand } from '@numen/plugin-sdk'

/** The words the bar writes itself. */
export const WORDS = {
  bar: 'Status bar',
  /** The row of a menu of items that are being read. */
  loadingText: 'Loading…',
} as const

/** The hover text of a command: its title and, where it has one, its shortcut. */
export function formatTooltip(command: SlotCommand): string {
  return command.shortcut ? `${command.title} (${command.shortcut})` : command.title
}
