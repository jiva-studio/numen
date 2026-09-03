/**
 * A stream followed for as long as the window is open.
 *
 * The stream ends when the core stops or the connection goes, and it is taken
 * up again after a wait. A window that had stopped following would look
 * exactly like one that is up to date, so what it lost touch with is said
 * until it has it back.
 */

/** How long the window waits before it takes a stream up again, in milliseconds. */
export const AGAIN = 1000

/** What following a stream reads of the window it is following for. */
export interface Follows {
  /** Whether the window is still open. Nothing is followed once it is not. */
  open(): boolean
  /** What the window lost touch with, said until it has it back. */
  lost(said: string): void
  wait(ms: number): Promise<unknown>
  /**
   * What the follower lets go of when a stream ends: whatever it holds answers
   * to that reading of the stream alone.
   */
  reset?(): void
}

export function following(deps: Follows) {
  /**
   * One stream, read for as long as the window is open. What arrives is
   * answered before the next of it is read, and what the answer throws ends
   * this reading of the stream and begins another.
   */
  return async function follows<Said>(
    stream: () => AsyncIterable<Said>,
    each: (said: Said) => void | Promise<void>,
    again = AGAIN,
  ): Promise<void> {
    while (deps.open()) {
      try {
        for await (const said of stream()) {
          if (!deps.open()) return
          await each(said)
        }
      } catch (error) {
        if (!deps.open()) return
        deps.lost(String(error))
      }
      deps.reset?.()
      await deps.wait(again)
      // Taken up again, so what was said about losing it no longer holds.
      if (deps.open()) deps.lost('')
    }
  }
}
