/**
 * The tabs of the window as the person works with them: the layout, the moves to other windows and
 * the list of the open tabs.
 */
import { createWindowLayout, startPlacedTabs, useOpenTabs, type WindowTabs } from '@/entities/tab'
import { isSecondaryWindow } from '@/entities/window'
import { createWindowTabMover } from '@/features/tab-move'
import type { Notes } from '@/shared/model/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { WindowApi } from './createWindowApi'

/** What the tabs of the window use of the tabs and the notes. */
export interface WindowTabPartsDeps {
  readonly api: Pick<WindowApi, 'tabMove' | 'savedTabs'>
  readonly windowTabs: WindowTabs
  readonly windowId: string
  readonly notes: Notes
  readonly writeMessage: MessageWriter
}

export function createWindowTabParts(deps: WindowTabPartsDeps) {
  const { api, windowTabs, windowId, notes, writeMessage } = deps
  const isSecondary = isSecondaryWindow()
  const layout = createWindowLayout(isSecondary)
  return {
    layout,
    isSecondary,
    moves: createWindowTabMover({
      api: api.tabMove,
      windowId,
      windowTabs,
      notes,
      closeTab: windowTabs.closeTab,
      writeMessage,
    }),
    openTabs: useOpenTabs({
      windowTabs,
      isWriting: layout.isWritingTabs,
      writeMessage,
      writeOpenTabs: api.savedTabs.writeOpenTabs,
    }),
    openCurrentVault: () => startPlacedTabs(windowTabs, writeMessage),
  }
}
