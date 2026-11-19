/**
 * A stream followed for as long as the window is open.
 *
 * The stream ends when the core stops or the connection drops, and it is
 * resumed after a delay. A window that stopped following would look identical
 * to one that is up to date, so the lost connection is reported until the
 * stream resumes.
 */
import { AGAIN, LOST } from './streamRetry'

/** What following a stream reads of the window it is following for. */
export interface StreamDeps {
  /** Whether the window is still open. Nothing is followed once it is not. */
  isOpen(): boolean
  /** Reports the lost connection, until the stream resumes. */
  setLost(message: string): void
  wait(ms: number): Promise<void>
  /**
   * What the follower releases when a stream ends: whatever it holds answers
   * to that reading of the stream alone.
   */
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

export function createFollower(deps: StreamDeps) {
  const disconnected = new Set<symbol>()

  return async function follow<T>(
    stream: () => AsyncIterable<T>,
    each: (item: T) => void | Promise<void>,
    again = AGAIN,
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
      if (disconnected.size === 0) deps.setLost(LOST)
      disconnected.add(subscription)
    }
    try {
      while (deps.isOpen()) {
        await readStream(stream, deliver, () => deps.isOpen()).catch(reportLoss)
        if (!deps.isOpen()) return
        deps.reset?.()
        await deps.wait(again)
      }
    } finally {
      disconnected.delete(subscription)
    }
  }
}
