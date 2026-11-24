/**
 * Structured application logs, as the frontends of the plugins watch and write them.
 */
import { createStreamReader, createLogClient } from '@numen/core-client'
import type { LogHost } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { WindowClients } from '@/shared/api/clients'

const wait = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

/**
 * Structured application logs over `logService`: each watch follows the stream of the core, taken
 * up again after it drops, until the watch is disposed, and each write records an entry under the
 * calling plugin's id.
 */
export const createLogHost = (
  logService: WindowClients['logService'],
): ((pluginId: string) => LogHost) => {
  const logClient = createLogClient(logService)
  return (pluginId: string) => ({
    getLevel: () => logClient.getLogLevel(),
    setLevel: (level) => logClient.setLogLevel(level),
    watch: (handler) => {
      const listening = new AbortController()
      const follower = createStreamReader({
        isOpen: () => !listening.signal.aborted,
        setLost: () => undefined,
        wait,
      })
      runInBackground(
        follower(
          () => logClient.watchLogs(listening.signal),
          (entries) => handler(entries),
        ),
        () => undefined,
      )
      return { dispose: () => listening.abort() }
    },
    write: (entry) =>
      logClient.writeLog({
        level: entry.level,
        source: pluginId,
        message: entry.message,
        properties: entry.properties,
      }),
  })
}
