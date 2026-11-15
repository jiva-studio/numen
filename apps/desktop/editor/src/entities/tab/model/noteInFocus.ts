import type { WindowTabs } from './windowTabs'

/**
 * The note the most recently active tool tab depends on, among the tools that follow a note: those
 * that navigate to a path and report the path they show. Empty where there is none.
 */
export function getNoteInFocus(windowTabs: Pick<WindowTabs, 'tabs'>): {
  readonly path: string
  readonly title: string
} {
  for (const entry of [...windowTabs.tabs.tabs.value].reverse()) {
    const one = windowTabs.tabs.getTab(entry.id)
    if (one?.kind.display?.isTool && one.kind.address?.navigate && one.kind.address.getTarget) {
      const target = one.kind.address.getTarget(one.state)
      return { path: target.path ?? '', title: target.title ?? '' }
    }
  }
  return { path: '', title: '' }
}
