import type { TitleBarCommand } from '@numen/plugin-sdk'

/** The title of the window: the application and the vault, or the application alone. */
export function formatTitle(appName: string, vaultName: string): string {
  const vault = vaultName.trim()
  return vault === '' ? appName : `${appName} — ${vault}`
}

/** The tooltip of an command: its title and, where it has one, its shortcut. */
export function formatTooltip(command: TitleBarCommand): string {
  return command.shortcut ? `${command.title} (${command.shortcut})` : command.title
}
