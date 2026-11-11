/**
 * The tabs of the window as the person works with them: the layout, the moves to other windows, the
 * list of the open tabs and the links between the tabs of the windows.
 */
import {
  createWindowLayout,
  startPlacedTabs,
  useOpenTabs,
  type TabLinkGroup,
  type TabPathSync,
  type WindowTabs,
} from '@/entities/tab'
import type { Notes } from '@/shared/model/command'
import { isSecondaryWindow } from '@/entities/window'
import {
  createWindowTabLinkSession,
  type TabLinksState,
  type TabReports,
} from '@/features/tab-linking'
import { createMoveAcceptor, createMovedTabStarter, createWindowTabMover } from '@/features/tab-move'
import type { MessageWriter } from '@/shared/model/notices/messages'

/** What the tabs of the window use of the tabs, the links and the notes. */
export interface WindowTabPartsDeps {
  readonly windowTabs: WindowTabs
  readonly windowId: string
  readonly tabLinks: TabLinksState
  readonly reports: TabReports
  readonly notes: Notes
  readonly tabSync: {
    readonly applySync: (sync: TabPathSync) => void
    readonly applyGroupPaths: (groups: readonly TabLinkGroup[]) => void
  }
  readonly writeMessage: MessageWriter
}

export function createWindowTabParts(deps: WindowTabPartsDeps) {
  const { windowTabs, windowId, tabLinks, reports, writeMessage } = deps
  const isSecondary = isSecondaryWindow()
  const layout = createWindowLayout(isSecondary)
  const moves = createWindowTabMover({
    windowId,
    windowTabs,
    notes: deps.notes,
    closeTab: windowTabs.closeTab,
    writeMessage,
  })
  const openTabs = useOpenTabs({ windowTabs, isWriting: layout.isWritingTabs, writeMessage })
  const openMovedTab = createMovedTabStarter({
    windowTabs,
    closeTab: windowTabs.closeTab,
    acceptMove: createMoveAcceptor(windowId, reports.waitForReports),
    writeMessage,
  })
  const linkSession = createWindowTabLinkSession({
    windowId,
    windowTabs,
    isMain: !isSecondary,
    tabLinks,
    reports,
    applySync: deps.tabSync.applySync,
    applyGroupPaths: deps.tabSync.applyGroupPaths,
    openMovedTab,
    writeMessage,
  })
  return {
    layout,
    moves,
    openTabs,
    openMovedTab,
    linkSession,
    isSecondary,
    openCurrentVault: () => startPlacedTabs(windowTabs, writeMessage),
  }
}
