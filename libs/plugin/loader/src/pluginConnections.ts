/**
 * The calls of a plugin through its connections, carried out over ConnectionCallService for the rows of
 * that plugin.
 */
import type { Caught, ConnectionCallClient, ConnectionCallFailure } from '@numen/core-client'
import type { ConnectionError, ConnectionHost, Result } from '@numen/plugin-sdk'

const readError = (failure: ConnectionCallFailure): ConnectionError => ({
  error: failure.errorCode,
  message: failure.detail,
  failures: failure.failures.map((one) => ({
    connectionName: one.connectionName,
    reason: one.reason,
  })),
})

const readCancellation = (error: Caught, signal: AbortSignal | undefined): ConnectionError => {
  if (signal?.aborted !== true) throw error
  return {
    error: 'canceled',
    message: error instanceof Error ? error.message : String(error),
    failures: [],
  }
}

/**
 * The connections a plugin calls through, with the plugin id the loader gives it. `subscribeChanges` is
 * the window's signal of a change that can change the rows; without it the host has no onChange.
 */
export const createPluginConnections = (
  client: ConnectionCallClient,
  pluginId: string,
  subscribeChanges?: (handler: () => void) => () => void,
): ConnectionHost => ({
  call: async function* (operation, payload, options) {
    try {
      const end: Result<Uint8Array, ConnectionCallFailure> = yield* client.call(
        { pluginId, operation, payload },
        options,
      )
      return end.ok ? end : { ok: false, error: readError(end.error) }
    } catch (error) {
      return { ok: false, error: readCancellation(error, options?.signal) }
    }
  },
  listRows: (operation) => client.listRows(pluginId, operation),
  listRowOptions: (request) => client.listRowOptions({ pluginId, ...request }),
  ...(subscribeChanges && {
    onChange: (handler: () => void) => ({ dispose: subscribeChanges(handler) }),
  }),
})
