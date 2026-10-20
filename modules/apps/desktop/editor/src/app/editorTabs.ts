import type { useWindowTabs } from '@/entities/tab'
import type { NoteLookup } from '@/features/command-palette'

/** The tab of a kind that edits files, standing at a path, and nothing where none does. */
export function findEditorTabAt(
  windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'tabs' | 'getTab'>,
  path: string,
): string | null {
  for (const tab of windowTabs.tabs.value) {
    const open = windowTabs.getTab(tab.id)
    if (open?.kind.flushPending && open.kind.getOpenTab?.(open.state)?.path === path) return tab.id
  }
  return null
}

/** What the window knows of the note open at a path: the tab it is in, and what that tab is called. */
export function createNoteLookup(
  editing: {
    readonly reached: { getTabAt(path: string): string | null }
    getTitle(id: string): string
  },
  windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'tabs' | 'getTab'>,
): NoteLookup {
  const getTabAt = (path: string): string | null =>
    editing.reached.getTabAt(path) ?? findEditorTabAt(windowTabs, path)
  return {
    getTitle: (path) => {
      const id = getTabAt(path)
      if (id === null) return ''
      return editing.getTitle(id) || windowTabs.tabs.value.find((one) => one.id === id)?.title || ''
    },
    getTabAt,
  }
}
