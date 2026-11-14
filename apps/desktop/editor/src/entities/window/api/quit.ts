/**
 * The window quitting: the request to write what is pending, and the report that it is written.
 */
import { FLUSH_RESULT, getNames, type FlushResultCode } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'
import type { FlushOutcome, QuitRequest } from '../lib/quit'
import { WINDOW } from './window'

/** What a client has left, as the schema names it. */
const flushResults = getNames<FlushOutcome, FlushResultCode>({
  [FLUSH_RESULT.UNSPECIFIED]: null,
  [FLUSH_RESULT.NOTHING]: 'nothing',
  [FLUSH_RESULT.WRITTEN]: 'written',
  [FLUSH_RESULT.UNSAVED]: 'unsaved',
})

/** The quit of the window over `windowService`. */
export const createQuitApi = (windowService: WindowClients['windowService']) => {
  /**
   * The window quitting, until the client aborts the signal. The stream opens with the token this
   * client replies under.
   */
  async function* watchQuit(signal: AbortSignal): AsyncGenerator<QuitRequest> {
    for await (const request of windowService.watchQuit({ window: WINDOW }, { signal })) {
      yield { token: request.token, flush: request.flush }
    }
  }

  /** Reports that this client has written all pending data. */
  const reportFlush = async (token: string, outcome?: 'written' | 'unsaved'): Promise<void> => {
    await windowService.reportFlush({
      window: WINDOW,
      token,
      result: flushResults[outcome ?? 'nothing'],
    })
  }

  return { watchQuit, reportFlush }
}

export type QuitApi = ReturnType<typeof createQuitApi>
