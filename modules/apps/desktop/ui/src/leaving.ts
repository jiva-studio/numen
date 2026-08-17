/**
 * Writing out what the window still owes, when the window goes.
 *
 * A tab saves once the typing stops, so at any moment the last seconds of work
 * are in a buffer here and nowhere else. The application asks for them over a
 * stream this page listens on for as long as it is drawn, and waits for the
 * answer.
 */
import type { Core } from './showing'

/** Somewhere unwritten work is held, which the quit waits for. */
export type Owing = () => Promise<unknown>

/**
 * Answering the application when it asks for what the window owes.
 *
 * Whatever holds unwritten work adds itself, and every one of them is written
 * before the answer goes back. A refusal is not a reason to keep the window
 * open: the person asked for it to go.
 */
export function leaving(core: Core, wait: (ms: number) => Promise<unknown> = sleep) {
  const owing = new Set<Owing>()
  let open = true
  const listening = new AbortController()

  /** Something the quit waits for, until what this answers with is called. */
  const holds = (one: Owing) => {
    owing.add(one)
    return () => owing.delete(one)
  }

  async function answer(token: string) {
    await Promise.allSettled([...owing].map((one) => one()))
    await core.flushed(token)
  }

  /**
   * Listen for the quit, taken up again the way following is. A window that
   * had stopped listening would look exactly like one with nothing to write.
   */
  async function start() {
    while (open) {
      try {
        for await (const said of core.quitting(listening.signal)) {
          if (!open) return
          if (said.flush) await answer(said.token)
        }
      } catch {
        if (!open) return
      }
      await wait(1000)
    }
  }

  return {
    holds,
    start,
    close: () => {
      open = false
      listening.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
