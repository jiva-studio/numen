import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { WindowTabs } from './windowTabs'

/** Opens the tab each placed kind declares; a kind whose tab fails is reported and left out. */
export const openPlacedTabs = (
  windowTabs: Pick<WindowTabs, 'restored'>,
  writeMessage: MessageWriter,
): Promise<void> =>
  windowTabs.restored.openPlacedTabs((kind, error) =>
    writeMessage(`The ${kind} tab did not open: ${formatError(error)}`, 'error'),
  )

/** Opens the placed tabs without waiting; a failure of the whole open is reported. */
export const startPlacedTabs = (
  windowTabs: Pick<WindowTabs, 'restored'>,
  writeMessage: MessageWriter,
): void =>
  runInBackground(openPlacedTabs(windowTabs, writeMessage), (error) =>
    writeMessage(`The tabs did not open: ${formatError(error)}`, 'error'),
  )
