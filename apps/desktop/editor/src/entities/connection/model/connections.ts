/**
 * The connections the person made and the kinds of connection the plugins provide, as the core holds
 * them. A value of a secret goes to the core once and is not kept here.
 */
import { shallowRef } from 'vue'
import {
  asFailure,
  asValue,
  formatErrorMessage,
  type ConnectionChange,
  type ConnectionClient,
  type ConnectionDraft,
  type ConnectionFailure,
  type ConnectionInfo,
  type ConnectionTypeInfo,
  type Result,
} from '@numen/core-client'
import type { MessageWriter } from '@/shared/notices/messages'

/** The words this module shows in the window. */
export interface ConnectionOptions {
  /** The reason a refused call is shown to the person, in words. */
  readonly describeFailure: (failure: ConnectionFailure) => string
  /** The start of the message shown when the core cannot be read. */
  readonly loadFailed: string
  readonly notify: MessageWriter
}

export function createConnectionStore(client: ConnectionClient, options: ConnectionOptions) {
  const { describeFailure, loadFailed, notify } = options
  const connections = shallowRef<readonly ConnectionInfo[]>([])
  const types = shallowRef<readonly ConnectionTypeInfo[]>([])
  let hasLoaded = false

  /** Reads the types and the connections again. A failed read keeps what the store held. */
  const load = async (): Promise<void> => {
    hasLoaded = true
    try {
      const [readTypes, readConnections] = await Promise.all([client.listTypes(), client.list()])
      types.value = readTypes
      connections.value = readConnections
      notify('')
    } catch (caughtError) {
      notify(`${loadFailed} ${formatErrorMessage(caughtError)}`, 'error')
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
      if (!answer.ok) return asFailure(describeFailure(answer.error))
      await load()
      return asValue(answer.value)
    } catch (caughtError) {
      return asFailure(formatErrorMessage(caughtError))
    }
  }

  return {
    connections,
    types,
    load,
    reload,
    create: (draft: ConnectionDraft) => run(() => client.create(draft)),
    update: (change: ConnectionChange) => run(() => client.update(change)),
    delete: (id: string) => run(() => client.delete(id)),
  }
}
