/**
 * What the window is showing, and the rules for changing it.
 */
import { ref, shallowRef } from 'vue'
import type { Core, PathRename, NoteEdit, Span, Task } from '../shared/core'
import { formatErrorMessage } from '@numen/wire'
import { useWindowStreams } from './streams'

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
  told?(paths: readonly string[], renamed?: readonly PathRename[]): void | Promise<void>
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

export function useWindowShowing(core: Core, how: ShowingOptions = {}) {
  const wait = how.wait ?? sleep
  const told = how.told ?? (() => {})
  const drawing = how.drawing ?? (() => {})
  const wanted = how.wanted ?? (() => {})
  const reads = how.reads ?? (() => {})
  const reloads = how.reloads ?? (() => {})
  const name = ref('')
  /** The folder the vault the window is showing sat in when the page was drawn. */
  const at = ref('')
  const isIndexing = ref(true)
  /** The core could not be reached: nothing else in the window is true. */
  const failure = ref('')
  /** What the window itself lost touch with, said until it has it back. */
  const lost = ref('')
  const error = ref('')
  const unwatched = ref('')
  /** Why an agent cannot be reached, as the vault last answered. */
  const unreachable = ref('')
  /** Whether the vault holds a note to show at all. */
  const hasNote = ref(false)
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
  const isEmbedding = ref(false)
  /**
   * Everything the application is doing behind the window. It arrives whole
   * and is shown whole, and a new kind of work is an entry here.
   */
  const tasks = shallowRef<readonly Task[]>([])

  let open = true
  /** Let go of every stream the window is listening to. */
  const listening = new AbortController()

  /** The note the vault opens with, and whether it holds one at all. */
  async function first() {
    const note = await core.opening()
    hasNote.value = note !== null
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
    error.value = state.scan.failureReason
    unwatched.value = state.scan.unwatchedPath
    chunks.value = Number(state.coverage.chunkCount)
    embedded.value = Number(state.coverage.embeddedCount)
    isEmbedding.value = state.coverage.isEmbedding
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

  const { follow, draw, watch, attend } = useWindowStreams({
    core,
    listening,
    isOpen: () => open,
    setLost: (said) => {
      lost.value = said
    },
    wait,
    opening,
    tasks,
    swapped,
    reloads,
    told,
    first,
    ask,
    drawing,
    wanted,
    reads,
  })

  /** Waits for the scan to have stored something, then shows the first note. */
  async function start() {
    try {
      while (open) {
        const state = await ask()
        const note = await first()
        if (!open) return
        if (note) {
          isIndexing.value = false
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
        if (state.scan.failureReason || state.scan.isReady) {
          isIndexing.value = false
          void follow()
          void watch()
          void draw()
          void attend()
          return
        }
        await wait(100)
      }
    } catch (error) {
      failure.value = formatErrorMessage(error)
      isIndexing.value = false
    }
  }

  return {
    name,
    isIndexing,
    indexing: isIndexing,
    failure,
    lost,
    error,
    unwatched,
    unreachable,
    hasNote,
    holds: hasNote,
    opening,
    first,
    chunks,
    embedded,
    isEmbedding,
    embedding: isEmbedding,
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
