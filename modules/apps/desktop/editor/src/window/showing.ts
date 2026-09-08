/**
 * What the window is showing, and the rules for changing it.
 *
 * Apart from the template because these are the rules that decide whether the
 * window keeps up with the vault, and a rule inside a component is a rule that
 * is only exercised by looking at the screen.
 */
import { ref, shallowRef } from 'vue'
import type { Core, Move, NoteEdit, Span, Task } from '../shared/core'
import { following } from '@numen/ui'
import { troubleWords } from '@numen/wire'

/**
 * The vault as the whole window reads it, and what it says when the vault
 * changes.
 *
 * One window reads the vault once: one stream of changes, one stream of edits,
 * one stream of notes asked for, one stream of what is being done, and one set
 * of counts. What is drawn from any of it, and by how many tabs, is the
 * window's; nothing here knows what a tab holds.
 */
/** What the window hands the reading of a vault, beside the vault itself. */
export interface ShowingOptions {
  wait?(ms: number): Promise<unknown>
  /**
   * What hears that the vault changed, and is waited for. A change carrying no
   * paths names nothing: everything showing the vault reads again.
   */
  told?(paths: readonly string[], renamed?: readonly Move[]): void | Promise<void>
  /** What hears about a change to a note while it is being made. */
  drawing?(said: NoteEdit): void
  /** What puts a note in front of the person, asked for from outside the window. */
  wanted?(path: string): void | Promise<void>
  /**
   * What opens a document at spans of its own text, in the tab it is read
   * in. The person is taken to the first of them.
   */
  reads?(path: string, spans: readonly Span[]): void
  /** What draws the page again, once another vault is under this window. */
  reloads?(): void
}

export function showing(core: Core, how: ShowingOptions = {}) {
  const wait = how.wait ?? sleep
  const told = how.told ?? (() => {})
  const drawing = how.drawing ?? (() => {})
  const wanted = how.wanted ?? (() => {})
  const reads = how.reads ?? (() => {})
  const reloads = how.reloads ?? (() => {})
  const name = ref('')
  /** The folder the vault the window is showing sat in when the page was drawn. */
  const at = ref('')
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
   * Everything the application is doing behind the window. It arrives whole
   * and is shown whole, and a new kind of work is an entry here.
   */
  const tasks = shallowRef<readonly Task[]>([])

  let open = true
  /** Let go of every stream the window is listening to. */
  const listening = new AbortController()
  /** Every stream is read the same way, and taken up again the same way. */
  const follows = following({
    open: () => open,
    lost: (said) => {
      lost.value = said
    },
    wait,
  })

  /** The note the vault opens with, and whether it holds one at all. */
  async function first() {
    const note = await core.opening()
    holds.value = note !== null
    opening.value = note?.path ?? ''
    return opening.value
  }

  /**
   * What the vault says about itself, which is not only its name. Whether an
   * agent can be reached is asked beside it: that answer is the installation's
   * and does not change when another vault opens here.
   */
  async function ask() {
    const [state, agent] = await Promise.all([core.state(), core.agentUnreachable()])
    unreachable.value = agent
    name.value = state.name
    // Read once: this is the folder the page was drawn on.
    if (at.value === '') at.value = state.path
    trouble.value = state.scan.failed
    unwatched.value = state.scan.unwatched
    chunks.value = Number(state.coverage.chunkCount)
    embedded.value = Number(state.coverage.embeddedCount)
    embedding.value = state.coverage.embedding
    return state
  }

  /** Whether the vault under this window stands at another folder than the page. */
  async function swapped() {
    const was = at.value
    try {
      const state = await ask()
      return was !== '' && state.path !== was
    } catch {
      // The next change asks again.
      return false
    }
  }

  /**
   * Follow the vault.
   *
   * Every change is told, including changes to notes that are nowhere on
   * screen: a link is written at one end and shows at both, so a note edited
   * somewhere else is exactly how a new parent arrives. What is drawn cannot
   * answer whether a change reaches it.
   */
  const follow = () =>
    follows(
      () => core.changes(listening.signal),
      async (change) => {
        if (change.paths.length === 0 && !change.reload && change.renamed.length === 0) return
        // A reload standing at another folder is another vault under this
        // window, and the page is drawn again on it.
        if (change.reload && (await swapped())) return void reloads()
        await told(change.reload ? [] : change.paths, change.renamed)
        // The note the vault opens with is asked for again when it moves.
        if (change.renamed.some((went) => went.from === opening.value)) {
          try {
            await first()
          } catch {
            // The next change asks again.
          }
        }
        try {
          await ask()
        } catch {
          // What a change means is already drawn; the counts come round with
          // the next one.
        }
      },
    )

  /** Follows the changes being made to notes. */
  const draw = () =>
    follows(
      () => core.editing(listening.signal),
      (said) => {
        // The stream opens by saying nothing, which is how an open one is told
        // from one that never opened.
        if (said.path) drawing(said)
      },
    )

  /**
   * Travels to whatever is asked for while the window is open — an agent
   * working the vault beside the person naming the note it is talking about. A
   * focus naming a span of a source's text opens that source at it.
   */
  const watch = () =>
    follows(
      () => core.focus(listening.signal),
      async (asked) => {
        if (!asked.path) return
        const spans = asked.spans
          .map((one) => ({ from: one.from, to: one.to }))
          .filter((one) => one.to > one.from)
        if (!spans.length) return void (await wanted(asked.path))
        reads(asked.path, spans)
      },
    )

  /**
   * Keeps the list of what is being done up to date while the window is open.
   *
   * Nothing is asked for on a timer. Work begins without the window: an agent
   * is told to read a document, and the list says so the moment it starts.
   */
  const attend = () =>
    follows(
      () => core.tasks(listening.signal),
      async (list) => {
        const ran = tasks.value.length > 0
        tasks.value = list
        // What the vault holds moves while a pass runs and settles when it
        // ends, so it is asked for again the moment the list empties.
        if (!ran || list.length > 0) return
        try {
          await ask()
        } catch {
          // The pass after this one asks again.
        }
      },
    )

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
        if (state.scan.failed || state.scan.ready) {
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
      failure.value = troubleWords(error)
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
