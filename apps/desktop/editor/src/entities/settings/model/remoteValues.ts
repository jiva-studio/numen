/**
 * Values held by another process, as a control shows them while they are written.
 *
 * `createRemoteValues` keeps, by key, the value a control requested: it is shown from the request
 * until a read that began after its write settled is taken, and a failed write shows the confirmed
 * value again. Writes go one batch at a time; values requested during a write are sent together
 * after it, the newest value of each key.
 */
import { shallowRef } from 'vue'
import { createWriteOrder } from '@numen/ui'
import type { Caught } from '@/shared/api/errors'

/** One read in flight: its place in the order of reads, and the writes when it began. */
type PendingRead = ReturnType<ReturnType<typeof createWriteOrder>['beginRead']>

interface Requested<V> {
  readonly token: number
  readonly value: V
  readonly settledAt: number | null
}

interface Waiter {
  readonly resolve: () => void
  readonly reject: (error: Caught) => void
}

export function createRemoteValues<V>(send: (values: ReadonlyMap<string, V>) => Promise<void>) {
  const order = createWriteOrder()
  const requested = shallowRef<ReadonlyMap<string, Requested<V>>>(new Map())
  let lastToken = 0
  let queued = new Map<string, { readonly token: number; readonly value: V }>()
  let waiters: Waiter[] = []
  let active: Promise<void> | null = null

  const settleBatch = (
    batch: ReadonlyMap<string, { readonly token: number }>,
    isWritten: boolean,
  ): void => {
    const next = new Map(requested.value)
    for (const [key, { token }] of batch) {
      const one = next.get(key)
      // Stryker disable next-line OptionalChaining: Map.get returns undefined in its type signature
      if (one?.token !== token) continue
      if (isWritten) next.set(key, { ...one, settledAt: order.getSettledWrites() })
      else next.delete(key)
    }
    requested.value = next
  }

  const sendBatch = async (): Promise<void> => {
    const batch = queued
    const batchWaiters = waiters
    queued = new Map()
    waiters = []
    const settle = order.beginWrite()
    try {
      await send(new Map([...batch].map(([key, one]) => [key, one.value])))
    } catch (error) {
      settle()
      settleBatch(batch, false)
      for (const one of batchWaiters) one.reject(error)
      return
    }
    settle()
    settleBatch(batch, true)
    for (const one of batchWaiters) one.resolve()
  }

  const drain = async (): Promise<void> => {
    while (queued.size > 0) await sendBatch()
    active = null
  }

  /** The value requested at a key and not yet read back, if any. */
  const getRequestedValue = (key: string): { readonly value: V } | undefined =>
    requested.value.get(key)

  /** Whether the write of the value requested at a key is still in flight. */
  const isPending = (key: string): boolean => requested.value.get(key)?.settledAt === null

  /**
   * Shows the values at once and sends them after the write in flight, if any. Resolves when the
   * batch that carried them is written, and rejects with its error where it failed.
   */
  const requestWrite = (values: ReadonlyMap<string, V>): Promise<void> => {
    // Stryker disable next-line UpdateOperator: token is an internal monotonic counter
    const token = ++lastToken
    const next = new Map(requested.value)
    for (const [key, value] of values) {
      next.set(key, { token, value, settledAt: null })
      queued.set(key, { token, value })
    }
    requested.value = next
    const completion = new Promise<void>((resolve, reject) => waiters.push({ resolve, reject }))
    active ??= drain()
    return completion
  }

  /** A read of every value begins. */
  const beginRead = (): PendingRead => order.beginRead()

  /**
   * Runs `apply` where no write overlapped the read and its answer is newer than every answer
   * taken, and then ends the requested values whose write settled before the read began. Returns
   * whether it was taken.
   */
  const takeRead = (pendingRead: PendingRead, apply: () => void): boolean => {
    if (!order.isReadAfterWrites(pendingRead) || !order.takeRead(pendingRead)) return false
    apply()
    const next = new Map(requested.value)
    for (const [key, one] of next) {
      // Stryker disable next-line ConditionalExpression: settledAt never exceeds settledWrites when isReadAfterWrites holds
      if (one.settledAt !== null && one.settledAt <= pendingRead.settledWrites) next.delete(key)
    }
    // Stryker disable next-line ConditionalExpression: skips redundant shallowRef assignment when no keys were removed
    if (next.size !== requested.value.size) requested.value = next
    return true
  }

  return {
    getRequestedValue,
    isPending,
    isWriteInFlight: (): boolean => active !== null,
    requestWrite,
    beginRead,
    takeRead,
  }
}
