/**
 * The window quitting: the request to write what is pending, and the report that it is written.
 */
import { FLUSH_RESULT, getCodesByWord, type FlushResultCode } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'
import type { FlushOutcome, QuitRequest } from '../lib/quit'
import { WINDOW } from './window'

/** What a client has left, as the schema names it. */
const flushResults = getCodesByWord<FlushOutcome, FlushResultCode>({
  [FLUSH_RESULT.UNSPECIFIED]: null,
  [FLUSH_RESULT.NOTHING]: 'nothing',
  [FLUSH_RESULT.WRITTEN]: 'written',
  [FLUSH_RESULT.UNSAVED]: 'unsaved',
})

/** The window answers the quit of the core: it writes what is pending and reports it. */
export const createQuitApi = (windowService: WindowClients['windowService']) => {
  /**
   * The window quitting, until the client aborts the signal. The stream opens with the token this
   * client replies under.
   */
  async function* watchWindowClose(signal: AbortSignal): AsyncGenerator<QuitRequest> {
    for await (const request of windowService.watchWindowClose({ windowId: WINDOW }, { signal })) {
      yield { token: request.token, flush: request.isFlushRequired }
    }
  }

  /** Reports that this client has written all pending data. */
  const reportPendingWork = async (
    token: string,
    outcome?: 'written' | 'unsaved',
  ): Promise<void> => {
    await windowService.reportPendingWork({
      windowId: WINDOW,
      token,
      result: flushResults[outcome ?? 'nothing'],
    })
  }

  return { watchWindowClose, reportPendingWork }
}

export type QuitApi = ReturnType<typeof createQuitApi>
