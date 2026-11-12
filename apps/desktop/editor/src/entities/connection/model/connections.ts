/**
 * The connections the person made and the kinds of connection the plugins provide, as the core holds
 * them. A value of a secret goes to the core once and is not kept here.
 */
import { shallowRef } from 'vue'
import type { Result } from '@/shared/lib/result'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import {
  createConnection,
  deleteConnection,
  listConnections,
  listConnectionTypes,
  updateConnection,
} from '../api/connections'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionFailure,
  ConnectionInfo,
  ConnectionTypeInfo,
  OperationTitles,
} from '../lib/types'
import { CONNECTION_WORDS } from '../words'

export function createConnectionStore(notify: MessageWriter) {
  const connections = shallowRef<readonly ConnectionInfo[]>([])
  const types = shallowRef<readonly ConnectionTypeInfo[]>([])
  const operationTitles = shallowRef<OperationTitles>({})
  let hasLoaded = false

  /** Reads the types and the connections again. A failed read keeps what the store held. */
  const load = async (): Promise<void> => {
    hasLoaded = true
    try {
      const [readTypes, readConnections] = await Promise.all([
        listConnectionTypes(),
        listConnections(),
      ])
      types.value = readTypes.types
      operationTitles.value = readTypes.operationTitles
      connections.value = readConnections
      notify('')
    } catch (caughtError) {
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
      if (!answer.ok) return { ok: false, error: CONNECTION_WORDS.describeFailure(answer.error) }
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
    create: (draft: ConnectionDraft) => run(() => createConnection(draft)),
    update: (change: ConnectionChange) => run(() => updateConnection(change)),
    delete: (id: string) => run(() => deleteConnection(id)),
  }
}
export type ConnectionStore = ReturnType<typeof createConnectionStore>
