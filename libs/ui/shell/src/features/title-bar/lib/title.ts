import type { TitleBarAction } from './types'

/** The title of the window: the application and the vault, or the application alone. */
export function formatTitle(appName: string, vaultName: string): string {
  const vault = vaultName.trim()
  return vault === '' ? appName : `${appName} — ${vault}`
}

/** The tooltip of an action: its title and, where it has one, its shortcut. */
export function formatTooltip(action: TitleBarAction): string {
  return action.shortcut ? `${action.title} (${action.shortcut})` : action.title
}
