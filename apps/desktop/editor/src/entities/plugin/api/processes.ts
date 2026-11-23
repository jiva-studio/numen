/**
 * The plugins that run, as the frontends of the plugins watch them.
 */
import { createFollower, createProcessClient } from '@numen/core-client'
import type { ProcessHost } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { WindowClients } from '@/shared/api/clients'

const wait = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

/**
 * The plugins that run, over `pluginProcessService`: each watch follows the stream of the core, taken up
 * again after it drops, until the watch is disposed.
 */
export const createProcessHost = (
  pluginProcessService: WindowClients['pluginProcessService'],
): ProcessHost => {
  const processClient = createProcessClient(pluginProcessService)
  return {
    watch: (handler) => {
      const listening = new AbortController()
      const follower = createFollower({
        isOpen: () => !listening.signal.aborted,
        setLost: () => undefined,
        wait,
      })
      // A follower catches every fault of the stream it reads, so nothing reaches the error
      // handler.
      runInBackground(
        follower(
          () => processClient.watchProcesses(listening.signal),
          (list) => handler(list.processes, list.time),
        ),
        () => undefined,
      )
      return { dispose: () => listening.abort() }
    },
  }
}
