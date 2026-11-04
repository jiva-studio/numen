/**
 * Formatting an open tab for detaching into a secondary window, and parsing the
 * initial tab parameter when that window starts.
 */
import type { WindowTab } from '@/shared/model/tabKinds'

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/** Formats an open tab into the kind and optional file path to open in a detached window. */
export function formatDetachedTab(
  getTab: (id: string) => WindowTab | null | undefined,
  id: string,
): string {
  const tab = getTab(id)
  if (!tab) return id
  const path = tab.kind.getOpenTab?.(tab.state)?.path ?? ''
  return path ? `${tab.kind.kind}:${path}` : tab.kind.kind
}

/** Parses the `tab` query parameter of a detached window into a tab kind and target path. */
export function parseInitialTab(initialTab: string): {
  readonly kind: string
  readonly target: string
} {
  const searchFrom = initialTab.startsWith('reader:') ? 'reader:'.length : 0
  const colonIdx = initialTab.indexOf(':', searchFrom)
  const kind = colonIdx !== -1 ? initialTab.slice(0, colonIdx) : initialTab
  const rawTarget = colonIdx !== -1 ? initialTab.slice(colonIdx + 1) : ''
  const target = UUID_PATTERN.test(rawTarget) || rawTarget === kind ? '' : rawTarget
  return { kind, target }
}
