/**
 * The moves of the tabs of one window: the tab sent to another window or a new one, and the outline
 * a window offers over itself to take a tab.
 */
import { runInBackground } from '@numen/ui'
import { getTabPaneRect } from '@numen/workbench'
import type { Notes } from '@/shared/model/command'
import type { WindowTabs } from '@/entities/tab'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { TabMoveApi } from '../api/tab-move'
import type { Point } from '../lib/move'
import { createTabMover } from './tabMove'

export interface WindowTabMoverDeps {
  readonly api: Pick<TabMoveApi, 'detachTab' | 'moveTab' | 'highlightWindow'>
  readonly windowId: string
  readonly windowTabs: Pick<WindowTabs, 'tabs' | 'panes'>
  readonly notes: Pick<Notes, 'asking' | 'flushPending'>
  readonly closeTab: (id: string) => boolean
  readonly writeMessage: MessageWriter
}

const ignoreError = (): undefined => undefined

export function createWindowTabMover(deps: WindowTabMoverDeps) {
  const mover = createTabMover({
    ...deps,
    getPaneRect: (id) => getTabPaneRect(deps.windowTabs.panes.layout.value, id),
  })
  // The core does not order requests, so each highlight waits for the previous one; an off
  // request then cannot arrive before the on request it ends.
  let lastHighlight: Promise<void> = Promise.resolve()
  return {
    windowId: deps.windowId,
    isTabAsking: (id: string): boolean => deps.notes.asking(id),
    highlightWindow: (windowId: string, isOn: boolean): void => {
      lastHighlight = lastHighlight
        .then(() => deps.api.highlightWindow(windowId, isOn))
        .catch(ignoreError)
    },
    detachTab: (id: string, point: Point): void =>
      runInBackground(mover.detachTab(id, point), ignoreError),
    detachTabInPlace: (id: string): void =>
      runInBackground(mover.detachTabInPlace(id), ignoreError),
    moveTab: (id: string, windowId: string): void =>
      runInBackground(mover.moveTab(id, windowId), ignoreError),
  }
}
