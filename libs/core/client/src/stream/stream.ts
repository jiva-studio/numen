import { CONNECTION_LOST_MESSAGE, RETRY_DELAY_MS } from './streamRetry'

export interface StreamDeps {
  /** Whether the window is still open. Nothing is followed once it is not. */
  isOpen(): boolean
  /** Reports the lost connection, until the stream resumes. */
  setLost(message: string): void
  wait(ms: number): Promise<void>
  /** Releases what the reader of a stream holds when the stream ends. */
  reset?(): void
}

async function readStream<T>(
  stream: () => AsyncIterable<T>,
  each: (item: T) => Promise<void>,
  isOpen: () => boolean,
): Promise<void> {
  for await (const item of stream()) {
    if (!isOpen()) return
    await each(item)
  }
}

/**
 * Reads a stream for as long as the window is open and resumes it after `retryDelayMs` when it ends.
 * The loss is reported through `setLost` until the stream delivers again.
 */
export function createStreamReader(deps: StreamDeps) {
  const disconnected = new Set<symbol>()

  return async function follow<T>(
    stream: () => AsyncIterable<T>,
    each: (item: T) => void | Promise<void>,
    retryDelayMs = RETRY_DELAY_MS,
  ): Promise<void> {
    const subscription = Symbol()
    const deliver = async (item: T) => {
      await each(item)
      if (deps.isOpen() && disconnected.delete(subscription) && disconnected.size === 0) {
        deps.setLost('')
      }
    }
    const reportLoss = () => {
      if (!deps.isOpen()) return
      if (disconnected.size === 0) deps.setLost(CONNECTION_LOST_MESSAGE)
      disconnected.add(subscription)
    }
    try {
      while (deps.isOpen()) {
        await readStream(stream, deliver, () => deps.isOpen()).catch(reportLoss)
        if (!deps.isOpen()) return
        deps.reset?.()
        await deps.wait(retryDelayMs)
      }
    } finally {
      disconnected.delete(subscription)
    }
  }
}
