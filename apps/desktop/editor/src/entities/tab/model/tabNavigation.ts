import type { fileOpeners } from './openers'
import type { WindowTabs } from './windowTabs'

/** What the navigation of notes between the tabs of the window uses of them. */
export interface TabNavigationDeps {
  readonly tabOpeners: Pick<ReturnType<typeof fileOpeners>, 'openFile'>
  readonly windowTabs: Pick<WindowTabs, 'tabs'>
  readonly tabSync: { setShownPath(tabId: string, path: string): void }
}

/** Where a note goes in the window: opened in the current tab, or followed by the tabs that stood on it. */
export function createTabNavigation(deps: TabNavigationDeps) {
  return {
    /** Opens the note in a tab of the current window. */
    travel: async (path: string): Promise<void> => {
      await deps.tabOpeners.openFile(path, '', 'current')
    },
    /** Every tab standing on a note moves to another note, and the tab sync records the path it shows. */
    leavePath: (from: string, to: string): Promise<void> =>
      deps.windowTabs.tabs.navigateTabsAt(from, to, (tabId) =>
        deps.tabSync.setShownPath(tabId, to),
      ),
  }
}
