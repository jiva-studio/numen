/**
 * Write queuing and dirty-field tracking for preset tabs.
 */
import { sendWrite, type WriteOptions, type WriteQueue } from './presetWrite'

const runWrite = async (queue: WriteQueue, options: WriteOptions): Promise<void> => {
  const settle = queue.inflight.order.beginWrite()
  try {
    await sendWrite(queue, options)
  } finally {
    settle()
  }
  queue.inflight.isWriting = false
  const again = queue.inflight.isWriteQueued && !queue.status.hasExternalChange.value
  const next = queue.inflight.queuedOptions ?? options
  queue.inflight.isWriteQueued = false
  queue.inflight.queuedOptions = null
  if (again) {
    await requestWrite(queue, next)
    return
  }
  queue.inflight.activeWrite = null
}

/** Requests a write, queuing behind the active write. */
export const requestWrite = (queue: WriteQueue, options: WriteOptions): Promise<void> => {
  if (queue.inflight.isWriting) {
    queue.inflight.isWriteQueued = true
    queue.inflight.queuedOptions = options
    return queue.inflight.activeWrite ?? Promise.resolve()
  }
  queue.inflight.isWriting = true
  const started = runWrite(queue, options)
  queue.inflight.activeWrite = started
  return started
}

/** Flushes any unwritten changes or waits for the active write to finish. */
export const flushWrites = async (queue: WriteQueue, options: WriteOptions): Promise<void> => {
  if (queue.unsaved.dirtyFields.size > 0 && !queue.status.hasExternalChange.value) {
    await requestWrite(queue, options)
    // Stryker disable next-line ConditionalExpression: Settled write is null and awaiting null resolves immediately.
  } else if (queue.inflight.activeWrite) {
    await queue.inflight.activeWrite
  }
}

/** Checks whether a tab can close, warning once if changes cannot be written. */
export const canCloseTab = async (queue: WriteQueue, options: WriteOptions): Promise<boolean> => {
  await flushWrites(queue, options)
  if (queue.unsaved.dirtyFields.size === 0 && !queue.status.hasExternalChange.value) return true
  if (queue.unsaved.hasWarned) return true
  queue.unsaved.hasWarned = true
  return false
}
