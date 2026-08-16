/**
 * What the window is showing, and the rules for changing it.
 *
 * Apart from the template because these are the rules that decide whether the
 * window keeps up with the vault, and a rule inside a component is a rule that
 * is only exercised by looking at the screen.
 */
import { ref } from 'vue'
import type { Neighbourhood } from './plex'

/** Everything the window asks of the core, and nothing about how it is drawn. */
export interface Core {
  neighbourhood(path: string): Promise<Neighbourhood>
  opening(): Promise<{ path: string } | null>
  state(): Promise<{ name: string; ready: boolean; failed: string; unwatched: string }>
  changes(): AsyncIterable<{ paths: string[]; reload: boolean }>
}

export function showing(core: Core, wait: (ms: number) => Promise<unknown> = sleep) {
  const neighbourhood = ref<Neighbourhood | null>(null)
  /**
   * The note the window is showing, as it asked for it.
   *
   * Kept apart from what came back: when the note is gone the answer carries no
   * path, and taking the next question from the focus would leave the window
   * with nothing to ask about and no way back.
   */
  const here = ref('')
  const name = ref('')
  const indexing = ref(true)
  /** The core could not be reached: nothing else in the window is true. */
  const failure = ref('')
  /** Something is wrong and the window still works: it says so and carries on. */
  const notice = ref('')
  const trouble = ref('')
  const unwatched = ref('')

  /**
   * Which question is the current one. Two answers can be in flight — a click
   * while a change is being followed — and without this the slower one wins
   * whatever was asked last.
   */
  let asked = 0
  let open = true

  async function go(path: string) {
    const mine = ++asked
    try {
      const answer = await core.neighbourhood(path)
      if (mine !== asked) return
      if (!answer.focus?.path) {
        // The vault no longer holds it. What is on screen stays, and following
        // goes on, so putting the file back brings it straight back.
        notice.value = `${path} is not in the vault`
        return
      }
      notice.value = ''
      here.value = path
      neighbourhood.value = answer
    } catch (error) {
      if (mine !== asked) return
      notice.value = String(error)
    }
  }

  /** What the vault says about itself, which is not only its name. */
  async function ask() {
    const state = await core.state()
    name.value = state.name
    trouble.value = state.failed
    unwatched.value = state.unwatched
    return state
  }

  /**
   * Follow the vault.
   *
   * Every change asks for the picture again, including changes to notes that
   * are nowhere on it: a link is written at one end and shows at both, so a
   * note edited somewhere else is exactly how a new parent arrives. What is on
   * screen cannot answer whether a change reaches it.
   *
   * The stream ends when the core stops or the connection goes, and it is taken
   * up again. A window that had stopped following would look exactly like one
   * that is up to date.
   */
  async function follow() {
    while (open) {
      try {
        for await (const change of core.changes()) {
          if (!open) return
          if (change.paths.length === 0 && !change.reload) continue
          if (here.value) {
            await go(here.value)
          } else {
            const note = await core.opening()
            if (note) await go(note.path)
          }
          await ask()
        }
      } catch (error) {
        if (!open) return
        notice.value = String(error)
      }
      await wait(1000)
    }
  }

  /** Waits for the scan to have stored something, then shows the first note. */
  async function start() {
    try {
      for (;;) {
        const state = await ask()
        const note = await core.opening()
        if (note) {
          indexing.value = false
          await go(note.path)
          void follow()
          return
        }
        // A vault that could not be read is not an empty one, and neither is
        // one still being read. Both end the waiting; only one is empty.
        if (state.failed || state.ready) {
          indexing.value = false
          void follow()
          return
        }
        await wait(100)
      }
    } catch (error) {
      failure.value = String(error)
      indexing.value = false
    }
  }

  return {
    neighbourhood,
    here,
    name,
    indexing,
    failure,
    notice,
    trouble,
    unwatched,
    go,
    start,
    follow,
    close: () => {
      open = false
    },
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
