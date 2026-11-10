/**
 * The places another component asked this window to display.
 */
import { workspace } from '@/shared/api/clients'
import type { FocusRequest } from '../lib/focus'

/** The requests to display a place, until the caller aborts the signal. */
export async function* watchFocus(signal: AbortSignal): AsyncGenerator<FocusRequest> {
  for await (const request of workspace.watchFocus({}, { signal })) {
    yield {
      path: request.path,
      ranges: request.ranges.map((one) => ({ start: one.start, end: one.end })),
      tabId: request.tabId,
    }
  }
}
