/**
 * Values held by another process, as a control shows them while they are written.
 *
 * `createRemoteValues` keeps, by key, the value a control requested: it is shown from the request
 * until the answer to its write is taken, and a failed write shows the confirmed value again. Every
 * answer of the other process carries the version of the state it stands for; an answer older than
 * one already taken is dropped. Writes go one batch at a time; values requested during a write are
 * sent together after it, the newest value of each key.
 */
import { shallowRef } from 'vue'
import type { Caught } from '@/shared/api/errors'

/** What a write answers: the version of the state it left. */
export interface WriteAnswer {
  readonly version: bigint
}

interface Requested<V> {
  readonly token: number
  readonly value: V
}

interface Waiter {
  readonly resolve: () => void
  readonly reject: (error: Caught) => void
}

export function createRemoteValues<V, A extends WriteAnswer>(
  send: (values: ReadonlyMap<string, V>) => Promise<A>,
  apply: (answer: A) => void,
) {
  const requested = shallowRef<ReadonlyMap<string, Requested<V>>>(new Map())
  let lastToken = 0
  let lastTakenVersion = 0n
  let queued = new Map<string, Requested<V>>()
  let waiters: Waiter[] = []
  let active: Promise<void> | null = null

  /** Ends the requests of a batch that no later request replaced. */
  const endBatch = (batch: ReadonlyMap<string, { readonly token: number }>): void => {
    const next = new Map(requested.value)
    for (const [key, { token }] of batch) {
      // Stryker disable next-line OptionalChaining: Map.get returns undefined in its type signature
      if (next.get(key)?.token === token) next.delete(key)
    }
    requested.value = next
  }

  /** Runs `show` where the answer is not older than every answer taken. Returns whether it ran. */
  const takeAnswer = (version: bigint, show: () => void): boolean => {
    if (version < lastTakenVersion) return false
    lastTakenVersion = version
    show()
    return true
  }

  const sendBatch = async (): Promise<void> => {
    const batch = queued
    const batchWaiters = waiters
    queued = new Map()
    waiters = []
    let answer: A
    try {
      answer = await send(new Map([...batch].map(([key, one]) => [key, one.value])))
    } catch (error) {
      endBatch(batch)
      for (const one of batchWaiters) one.reject(error)
      return
    }
    takeAnswer(answer.version, () => apply(answer))
    endBatch(batch)
    for (const one of batchWaiters) one.resolve()
  }

  const drain = async (): Promise<void> => {
    while (queued.size > 0) await sendBatch()
    active = null
  }

  /** The value requested at a key whose write has not answered, if any. */
  const getRequestedValue = (key: string): { readonly value: V } | undefined =>
    requested.value.get(key)

  /** Whether the write of the value requested at a key has not answered. */
  const isPending = (key: string): boolean => requested.value.has(key)

  /**
   * Shows the values at once and sends them after the write in flight, if any. Resolves when the
   * batch that carried them is written, and rejects with its error where it failed.
   */
  const requestWrite = (values: ReadonlyMap<string, V>): Promise<void> => {
    // Stryker disable next-line UpdateOperator: token is an internal monotonic counter
    const token = ++lastToken
    const next = new Map(requested.value)
    for (const [key, value] of values) {
      next.set(key, { token, value })
      queued.set(key, { token, value })
    }
    requested.value = next
    const completion = new Promise<void>((resolve, reject) => waiters.push({ resolve, reject }))
    active ??= drain()
    return completion
  }

  return { getRequestedValue, isPending, requestWrite, takeAnswer }
}
