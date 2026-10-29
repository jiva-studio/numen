/**
 * A stream followed for as long as the window is open.
 *
 * The stream ends when the core stops or the connection drops, and it is
 * resumed after a delay. A window that stopped following would look identical
 * to one that is up to date, so the lost connection is reported until the
 * stream resumes.
 */

/** How long the window waits before it takes a stream up again, in milliseconds. */
export const AGAIN = 1000

/**
 * The message reported when a stream ended in a fault.
 *
 * The fault itself belongs to the transport and is not actionable. What is
 * actionable is that the window is behind the vault and is retrying. Every
 * stream ends and resumes the same way, so one message serves them all and
 * this library needs no knowledge of what is being followed.
 */
export const LOST = 'lost touch with numen — the window keeps trying'

/** What following a stream reads of the window it is following for. */
export interface StreamDeps {
  /** Whether the window is still open. Nothing is followed once it is not. */
  isOpen(): boolean
  /** Reports the lost connection, until the stream resumes. */
  setLost(message: string): void
  wait(ms: number): Promise<unknown>
  /**
   * What the follower lets go of when a stream ends: whatever it holds answers
   * to that reading of the stream alone.
   */
  reset?(): void
}

export function createFollower(deps: StreamDeps) {
  /**
   * One reading of a stream, to its end or to the fault that ends it. It
   * answers whether the window is still open.
   */
  const readStream = async <T>(
    stream: () => AsyncIterable<T>,
    each: (item: T) => void | Promise<void>,
  ): Promise<boolean> => {
    try {
      for await (const item of stream()) {
        if (!deps.isOpen()) return false
        await each(item)
      }
    } catch {
      // Any error thrown while reading ends this reading and is reported as a
      // lost connection. This also covers a fault thrown by `each`, which cannot
      // be distinguished from a transport fault here.
      if (!deps.isOpen()) return false
      deps.setLost(LOST)
    }
    return true
  }

  /**
   * One stream, read for as long as the window is open. What arrives is
   * answered before the next of it is read, and what the answer throws ends
   * this reading of the stream and begins another.
   */
  return async function follows<T>(
    stream: () => AsyncIterable<T>,
    each: (item: T) => void | Promise<void>,
    again = AGAIN,
  ): Promise<void> {
    while (deps.isOpen()) {
      if (!(await readStream(stream, each))) return
      deps.reset?.()
      await deps.wait(again)
      // The stream resumed, so the lost-connection report is cleared.
      if (deps.isOpen()) deps.setLost('')
    }
  }
}
