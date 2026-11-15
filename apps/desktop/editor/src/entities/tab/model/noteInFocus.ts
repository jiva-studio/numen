import type { WindowTabs } from './windowTabs'

/**
 * The note the most recently active tool tab depends on, among the tools that follow a note: those
 * that navigate to a path and report the path they show. Empty where there is none.
 */
export function getNoteInFocus(windowTabs: Pick<WindowTabs, 'tabs'>): {
  readonly path: string
  readonly title: string
} {
  const last = [...windowTabs.tabs.tabs.value]
    .reverse()
    .map((one) => windowTabs.tabs.getTab(one.id))
    .find(
      (one) =>
        one?.kind.display?.isTool &&
        one.kind.address?.navigate !== undefined &&
        one.kind.address.getTarget !== undefined,
    )
  const target = last?.kind.address?.getTarget?.(last.state)
  return { path: target?.path ?? '', title: target?.title ?? '' }
}
