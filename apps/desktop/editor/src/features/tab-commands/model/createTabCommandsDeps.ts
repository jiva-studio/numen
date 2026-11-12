/**
 * What the commands over a tab reach in the window: the tabs it holds and the tabs of the others.
 */
import type { WindowTabs } from '@/entities/tab'
import type { TabCommandsDeps } from './commands'

/** The window and the tabs the commands over a tab read. */
export interface TabCommandsParts {
  readonly tabLinks: TabCommandsDeps['tabLinks']
  readonly windowTabs: Pick<WindowTabs, 'tabs'>
  readonly moves: {
    readonly windowId: string
    readonly isTabAsking: TabCommandsDeps['isAsking']
    readonly detachTabInPlace: TabCommandsDeps['detach']
    readonly moveTab: TabCommandsDeps['move']
  }
}

export function createTabCommandsDeps(parts: TabCommandsParts): TabCommandsDeps {
  const { tabLinks, windowTabs, moves } = parts
  return {
    tabLinks,
    getTabs: () => windowTabs.tabs.tabs.value,
    getKind: (id) => windowTabs.tabs.getTab(id)?.kind.kind,
    isLinkable: (id) => windowTabs.tabs.getTab(id)?.kind.address?.navigate !== undefined,
    windowId: moves.windowId,
    isAsking: moves.isTabAsking,
    detach: moves.detachTabInPlace,
    move: moves.moveTab,
  }
}
