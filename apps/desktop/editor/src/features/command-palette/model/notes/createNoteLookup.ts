import type { WindowTabs } from '@/entities/tab'
import type { NoteLookup } from '@/features/command-palette'

function findEditorTabAt(windowTabs: Pick<WindowTabs, 'tabs'>, path: string): string | null {
  for (const tab of windowTabs.tabs.tabs.value) {
    const open = windowTabs.tabs.getTab(tab.id)
    if (
      open?.kind.events?.flushPending &&
      open.kind.address?.getOpenTab?.(open.state).path === path
    )
      return tab.id
  }
  return null
}

/** What the window records for the note open at a path: the tab it is in, and the tab's title. */
export function createNoteLookup(
  editing: {
    readonly reached: { getTabAt(path: string): string | null }
    getTitle(id: string): string
  },
  windowTabs: Pick<WindowTabs, 'tabs'>,
): NoteLookup {
  const getTabAt = (path: string): string | null =>
    editing.reached.getTabAt(path) ?? findEditorTabAt(windowTabs, path)
  return {
    getTitle: (path) => {
      const id = getTabAt(path)
      if (id === null) return ''
      return (
        editing.getTitle(id) || windowTabs.tabs.tabs.value.find((one) => one.id === id)?.title || ''
      )
    },
    getTabAt,
  }
}
