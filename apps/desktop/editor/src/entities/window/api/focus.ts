/**
 * The places another component asked this window to display.
 */
import type { WindowClients } from '@/shared/api/clients'
import type { FocusRequest } from '../lib/focus'

/** The focus requests of the window over `workspace`. */
export const createFocusApi = (workspace: WindowClients['workspace']) => {
  /** The requests to display a place, until the caller aborts the signal. */
  async function* watchFocus(signal: AbortSignal): AsyncGenerator<FocusRequest> {
    for await (const request of workspace.watchFocus({}, { signal })) {
      yield {
        path: request.path,
        ranges: request.ranges.map((one) => ({ start: one.start, end: one.end })),
        tabId: request.tabId,
      }
    }
  }
  return { watchFocus }
}

export type FocusApi = ReturnType<typeof createFocusApi>
