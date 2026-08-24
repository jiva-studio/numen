/**
 * What the window is showing, and the rules for changing it.
 *
 * Apart from the template because these are the rules that decide whether the
 * window keeps up with the vault, and a rule inside a component is a rule that
 * is only exercised by looking at the screen.
 */
import { ref } from 'vue'
import type { Core, Run, Said, Task, Went } from './core'

/**
 * The vault as the whole window reads it, and what it says when the vault
 * changes.
 *
 * One window reads the vault once: one stream of changes, one stream of edits,
 * one stream of notes asked for, one stream of what is being done, and one set
 * of counts. What is drawn from any of it, and by how many tabs, is the
 * window's; nothing here knows what a tab holds.
 */
export function showing(
  core: Core,
  wait: (ms: number) => Promise<unknown> = sleep,
  /**
   * What hears that the vault changed, and is waited for. A change carrying no
   * paths names nothing: everything showing the vault reads again.
   */
  told: (
    paths: readonly string[],
    renamed?: readonly Went[],
  ) => void | Promise<void> = () => {},
  /** What hears about a change to a note while it is being made. */
  drawing: (said: Said) => void = () => {},
  /** What puts a note in front of the person, asked for from outside the window. */
  wanted: (path: string) => void | Promise<void> = () => {},
  /**
   * What opens a document at stretches of its own text, in the tab it is read
   * in. The person is taken to the first of them.
   */
  reads: (path: string, runs: readonly Run[]) => void = () => {},
) {
  const name = ref('')
  const indexing = ref(true)
  /** The core could not be reached: nothing else in the window is true. */
  const failure = ref('')
  /** What the window itself lost touch with, said until it has it back. */
  const lost = ref('')
  const trouble = ref('')
  const unwatched = ref('')
  /** Why an agent cannot be reached, as the vault last answered. */
  const unreachable = ref('')
  /** Whether the vault holds a note to show at all. */
  const holds = ref(false)
  /** The note the vault opens with, for whatever has nowhere else to start. */
  const opening = ref('')
  /**
   * How much of the text the index holds carries a vector.
   *
   * The whole of what the vault holds, which is what says whether it can be
   * searched by meaning at all. How far a pass has got is a task.
   */
  const chunks = ref(0)
  const embedded = ref(0)
  /**
   * Whether anything is going to embed what was cut.
   *
   * False is the ordinary state of an installation with no model: the vault is
   * searched by its words, and the count of embedded chunks stays where it is.
   */
  const embedding = ref(false)
  /**
   * Everything the application is doing behind the window.
   *
   * It arrives whole and is shown whole. A new kind of work is an entry here
   * rather than another count to read out of the state and another branch in
   * what draws it.
   */
  const tasks = ref<readonly Task[]>([])

  let open = true
  /** Let go of every stream the window is listening to. */
  const listening = new AbortController()

  /** The note the vault opens with, and whether it holds one at all. */
  async function first() {
    const note = await core.opening()
    holds.value = note !== null
    opening.value = note?.path ?? ''
    return opening.value
  }

  /** What the vault says about itself, which is not only its name. */
  async function ask() {
    const state = await core.state()
    name.value = state.name
    trouble.value = state.failed
    unwatched.value = state.unwatched
    unreachable.value = state.unreachable
    chunks.value = Number(state.chunks)
    embedded.value = Number(state.embedded)
    embedding.value = state.embedding
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
        for await (const change of core.changes(listening.signal)) {
          if (!open) return
          if (change.paths.length === 0 && !change.reload && change.renamed.length === 0) continue
          await told(change.reload ? [] : change.paths, change.renamed)
          try {
            await ask()
          } catch {
            // The stream stays open. What a change means is already drawn; the
            // counts come round with the next one.
          }
        }
      } catch (error) {
        if (!open) return
        lost.value = String(error)
      }
      await wait(1000)
      // Taken up again, so what was said about losing it no longer holds.
      if (open) lost.value = ''
    }
  }

  /** Follows the changes being made to notes, and takes the stream up again. */
  async function draw() {
    while (open) {
      try {
        for await (const said of core.editing(listening.signal)) {
          if (!open) return
          // The stream opens by saying nothing, which is how an open one is
          // told from one that never opened.
          if (said.path) drawing(said)
        }
      } catch (error) {
        if (!open) return
        lost.value = String(error)
      }
      await wait(1000)
    }
  }

  /**
   * Travels to whatever is asked for while the window is open — an agent
   * working the vault beside the person naming the note it is talking about. A
   * focus naming a stretch of a source's text opens that source at it.
   *
   * Taken up again the way following is, and for the same reason.
   */
  async function watch() {
    while (open) {
      try {
        for await (const asked of core.focus(listening.signal)) {
          if (!open) return
          if (!asked.path) continue
          if (asked.length) {
            const also = (asked.also ?? [])
              .filter((one) => (one.length ?? 0) > 0)
              .map((one) => ({ start: one.start ?? 0, length: one.length ?? 0 }))
            reads(asked.path, [{ start: asked.start ?? 0, length: asked.length }, ...also])
          } else await wanted(asked.path)
        }
      } catch (error) {
        if (!open) return
        lost.value = String(error)
      }
      await wait(1000)
    }
  }

  /**
   * Keeps the list of what is being done up to date while the window is open.
   *
   * Nothing is asked for on a timer. Work begins without the window: an agent is
   * told to read a document, and the list says so the moment it starts.
   *
   * Taken up again the way following is, and for the same reason.
   */
  async function attend() {
    while (open) {
      try {
        for await (const list of core.tasks(listening.signal)) {
          if (!open) return
          const ran = tasks.value.length > 0
          tasks.value = list
          // What the vault holds moves while a pass runs and settles when it
          // ends, so it is asked for again the moment the list empties.
          if (ran && list.length === 0) {
            try {
              await ask()
            } catch {
              // The stream stays open, and the pass after this one asks again.
            }
          }
        }
      } catch (error) {
        if (!open) return
        lost.value = String(error)
      }
      await wait(1000)
      // Taken up again, so what was said about losing it no longer holds.
      if (open) lost.value = ''
    }
  }

  /** Waits for the scan to have stored something, then shows the first note. */
  async function start() {
    try {
      while (open) {
        const state = await ask()
        const note = await first()
        if (!open) return
        if (note) {
          indexing.value = false
          // The vault holds a note to show: everything showing it reads now.
          await told([], [])
          void follow()
          void watch()
          void draw()
          void attend()
          return
        }
        // A vault that could not be read is not an empty one, and neither is
        // one still being read. Both end the waiting; only one is empty.
        if (state.failed || state.ready) {
          indexing.value = false
          void follow()
          void watch()
          void draw()
          void attend()
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
    name,
    indexing,
    failure,
    lost,
    trouble,
    unwatched,
    unreachable,
    holds,
    opening,
    first,
    chunks,
    embedded,
    embedding,
    tasks,
    start,
    follow,
    watch,
    close: () => {
      open = false
      listening.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
