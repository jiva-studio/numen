/** Whether a key event is the shortcut a command declares, such as `ctrl+shift+k`. */
export function matchesShortcut(event: KeyboardEvent, shortcut?: string): boolean {
  if (!shortcut) return false
  const parts = shortcut
    .toLowerCase()
    .split('+')
    .map((part) => part.trim())
  const key = parts[parts.length - 1]
  const hasMod =
    parts.includes('ctrl') ||
    parts.includes('control') ||
    parts.includes('mod') ||
    parts.includes('cmd')
  const hasAlt = parts.includes('alt') || parts.includes('option')
  const hasShift = parts.includes('shift')

  if (hasMod !== (event.ctrlKey || event.metaKey)) return false
  if (hasAlt !== event.altKey) return false
  if (hasShift !== event.shiftKey) return false
  return event.key.toLowerCase() === key
}
