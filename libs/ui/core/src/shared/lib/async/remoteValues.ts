/**
 * Values held by another process, as a control shows them while they are written.
 *
 * `createWriteOrder` orders the reads and writes of one remote store. `createRemoteValues` keeps,
 * by key, the value a control requested: it is shown from the request until a read that began after
 * its write settled is taken and shows the value, and a failed write shows the confirmed value again.
 * A caller that can tell whether a read shows the value keeps it past reads that do not, up to
 * `MAX_UNCONFIRMED_READS` of them, so a remote that answers with the old value for a moment does not
 * draw it. Writes go one
 * batch at a time; values requested during a write are sent together after it, the newest value of
 * each key.
 */
import { shallowRef } from 'vue'

/** One read in flight: its place in the order of reads, and the writes when it began. */
export interface PendingRead {
  readonly place: number
  readonly settledWrites: number
  readonly isWriting: boolean
}

export type WriteOrder = ReturnType<typeof createWriteOrder>

export function createWriteOrder() {
  let writesInFlight = 0
  let settledWrites = 0
  let lastRead = 0
  let lastTaken = 0

  /** A write begins; the function returned settles it, whatever its answer. */
  const beginWrite = (): (() => void) => {
    writesInFlight += 1
    let isSettled = false
    return () => {
      if (isSettled) return
      isSettled = true
      writesInFlight -= 1
      settledWrites += 1
    }
  }

  const isWriteInFlight = (): boolean => writesInFlight > 0

  /** The number of writes settled so far. */
  const getSettledWrites = (): number => settledWrites

  const beginRead = (): PendingRead => ({
    place: ++lastRead,
    settledWrites,
    isWriting: writesInFlight > 0,
  })

  /** Whether no write was in flight while the read ran: its answer cannot predate a write. */
  const isReadAfterWrites = (pendingRead: PendingRead): boolean =>
    !pendingRead.isWriting && writesInFlight === 0 && pendingRead.settledWrites === settledWrites

  /** Whether the answer of the read is newer than every answer taken; a newer one is taken. */
  const takeRead = (pendingRead: PendingRead): boolean => {
    if (pendingRead.place < lastTaken) return false
    lastTaken = pendingRead.place
    return true
  }

  return { beginWrite, isWriteInFlight, getSettledWrites, beginRead, isReadAfterWrites, takeRead }
}

/** The reads after a write that may show the old value before the requested value is let go. */
export const MAX_UNCONFIRMED_READS = 6

interface Requested<V> {
  readonly token: number
  readonly value: V
  /** The settled writes once this value was written; `null` while it is in flight. */
  readonly settledAt: number | null
  /** The reads taken since the write settled that did not show the value. */
  readonly unconfirmedReads: number
}

/**
 * The requested values after a read: those whose write settled before the read began end where the
 * read shows them, or after `MAX_UNCONFIRMED_READS` reads that do not. The same map where none ends
 * or counts the read.
 */
function releaseSettled<V>(
  requested: ReadonlyMap<string, Requested<V>>,
  settledWrites: number,
  isShown: ((key: string, value: V) => boolean) | undefined,
): ReadonlyMap<string, Requested<V>> {
  const next = new Map(requested)
  let isChanged = false
  for (const [key, one] of next) {
    if (one.settledAt === null || one.settledAt > settledWrites) continue
    isChanged = true
    const unconfirmedReads = one.unconfirmedReads + 1
    if (
      isShown === undefined ||
      isShown(key, one.value) ||
      unconfirmedReads >= MAX_UNCONFIRMED_READS
    ) {
      next.delete(key)
    } else {
      next.set(key, { ...one, unconfirmedReads })
    }
  }
  return isChanged ? next : requested
}

interface Waiter {
  readonly resolve: () => void
  readonly reject: (error: unknown) => void
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
  const getRequested = (key: string): { readonly value: V } | undefined => requested.value.get(key)

  /**
   * Shows the values at once and sends them after the write in flight, if any. Resolves when the
   * batch that carried them is written, and rejects with its error where it failed.
   */
  const requestWrite = (values: ReadonlyMap<string, V>): Promise<void> => {
    const token = ++lastToken
    const next = new Map(requested.value)
    for (const [key, value] of values) {
      next.set(key, { token, value, settledAt: null, unconfirmedReads: 0 })
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
   * Runs `apply` where the answer of the read is newer than every answer taken, and then ends the
   * requested values whose write settled before the read began and that the read shows, as
   * `isShown` tells after `apply`; without it every such value ends. A value the read does not show
   * ends after `MAX_UNCONFIRMED_READS` reads. Returns whether it was taken.
   */
  const takeRead = (
    pendingRead: PendingRead,
    apply: () => void,
    isShown?: (key: string, value: V) => boolean,
  ): boolean => {
    if (!order.takeRead(pendingRead)) return false
    apply()
    const next = releaseSettled(requested.value, pendingRead.settledWrites, isShown)
    if (next !== requested.value) requested.value = next
    return true
  }

  return {
    getRequested,
    isWriteInFlight: (): boolean => active !== null,
    requestWrite,
    beginRead,
    takeRead,
  }
}
