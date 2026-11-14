/**
 * The connections the person made and the kinds of connection the plugins provide, as the core holds
 * them. A value of a secret goes to the core once and is not kept here.
 */
import { shallowRef } from 'vue'
import { createRequestGuard } from '@numen/ui'
import type { Result } from '@/shared/lib/result'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { ConnectionApi } from '../api/connections'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionFailure,
  ConnectionInfo,
  ConnectionTypeInfo,
  OperationTitles,
} from '../lib/types'
import { CONNECTION_WORDS } from '../words'

export function createConnectionStore(api: ConnectionApi, notify: MessageWriter) {
  const connections = shallowRef<readonly ConnectionInfo[]>([])
  const types = shallowRef<readonly ConnectionTypeInfo[]>([])
  const operationTitles = shallowRef<OperationTitles>({})
  let hasLoaded = false
  const guard = createRequestGuard()

  /**
   * Reads the types and the connections again. A failed read keeps what the store held; the answer
   * to a read started before another is dropped.
   */
  const load = async (): Promise<void> => {
    hasLoaded = true
    const mine = guard.acquireRequest()
    try {
      const [readTypes, readConnections] = await Promise.all([
        api.listConnectionTypes(),
        api.listConnections(),
      ])
      if (!mine.isCurrent) return
      types.value = readTypes.types
      operationTitles.value = readTypes.operationTitles
      connections.value = readConnections
      notify('')
    } catch (caughtError) {
      if (!mine.isCurrent) return
      notify(`${CONNECTION_WORDS.loadFailed} ${formatError(caughtError)}`, 'error')
    }
  }

  /** Reads again where a page has read before: a window that never showed the connections reads none. */
  const reload = async (): Promise<void> => {
    if (hasLoaded) await load()
  }

  const run = async <T>(
    operation: () => Promise<Result<T, ConnectionFailure>>,
  ): Promise<Result<T, string>> => {
    try {
      const answer = await operation()
      if (!answer.ok) {
        if (answer.error.errorCode === 'stale_revision') await load()
        return { ok: false, error: CONNECTION_WORDS.describeFailure(answer.error) }
      }
      await load()
      return { ok: true, value: answer.value }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  return {
    connections,
    types,
    operationTitles,
    load,
    reload,
    create: (draft: ConnectionDraft) => run(() => api.createConnection(draft)),
    update: (change: ConnectionChange) => run(() => api.updateConnection(change)),
    delete: (id: string) => run(() => api.deleteConnection(id)),
    listRowOptions: api.listRowOptions,
  }
}
export type ConnectionStore = ReturnType<typeof createConnectionStore>
