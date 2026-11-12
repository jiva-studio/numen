import { runInBackground } from '@numen/ui'
import type { MovedTab } from '@/entities/tab'
import { formatError, type Caught } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { createMovedTabOpener, type MovedTabOpenerDeps } from './tabMove'

/** What a tab sent from another window is opened with; the failures are written as messages. */
export interface MovedTabStarterDeps extends Omit<MovedTabOpenerDeps, 'onError'> {
  readonly writeMessage: MessageWriter
}

/** Opens a tab sent from another window without waiting; a failure is written as a message. */
export function createMovedTabStarter(deps: MovedTabStarterDeps): (moved: MovedTab) => void {
  const report = (error: Caught) =>
    deps.writeMessage(`The moved tab did not open: ${formatError(error)}`, 'error')
  const open = createMovedTabOpener({
    windowTabs: deps.windowTabs,
    closeTab: deps.closeTab,
    acceptMove: deps.acceptMove,
    ...(deps.focusWindow ? { focusWindow: deps.focusWindow } : {}),
    ...(deps.wait ? { wait: deps.wait } : {}),
    onError: report,
  })
  return (moved) => runInBackground(open(moved), report)
}
