/**
 * The plugins that run, as the frontends of the plugins watch them.
 */
import { createProcessClient } from '@numen/core-client'
import type { ProcessHost } from '@numen/plugin-loader'
import { createFollower, runInBackground } from '@numen/ui'
import { pluginService } from '@/shared/api/clients'

const processes = createProcessClient(pluginService)

const wait = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

/**
 * The plugins that run: each watch follows the stream of the core, taken up again after it drops,
 * until the watch is disposed.
 */
export const processHost: ProcessHost = {
  watch: (handler) => {
    const listening = new AbortController()
    const follows = createFollower({
      isOpen: () => !listening.signal.aborted,
      setLost: () => undefined,
      wait,
    })
    // A follower catches every fault of the stream it reads, so nothing reaches the error handler.
    runInBackground(
      follows(() => processes.watchProcesses(listening.signal), handler),
      () => undefined,
    )
    return { dispose: () => listening.abort() }
  },
}
