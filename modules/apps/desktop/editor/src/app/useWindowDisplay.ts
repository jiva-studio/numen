/**
 * What the window is showing, and the rules for changing it.
 */
import { ref, shallowRef } from 'vue'
import type { NotePort } from '@/app/ports/notes'
import type { VaultPort } from '@/app/ports/vault'
import type { NoteEdit } from '@/entities/note'
import type { Task } from '@/shared/notices/task'
import type { PathRename } from '@/shared/paths'
import type { Span } from '@/shared/span'
import { formatErrorMessage } from '@numen/wire'
import { useWindowStreams } from './useWindowStreams'

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
export interface DisplayOptions {
  wait?(ms: number): Promise<unknown>
  /**
   * What hears that the vault changed, and is waited for. A change carrying no
   * paths is empty: everything showing the vault reads again.
   */
  onVaultChanged?(paths: readonly string[], renamed?: readonly PathRename[]): void | Promise<void>
  /** What hears about a change to a note while it is being made. */
  reportChange?(said: NoteEdit): void
  /** What puts a note in front of the person, asked for from outside the window. */
  travelTo?(path: string): void | Promise<void>
  /**
   * What opens a document at spans of its own text, in the tab it is read
   * in. The person is taken to the first of them.
   */
  openFileAt?(path: string, spans: readonly Span[]): void
  /** What draws the page again, once another vault is under this window. */
  reload?(): void
}

function createDisplayState() {
  return {
    name: ref(''),
    /** The folder the vault the window is showing sat in when the page was drawn. */
    at: ref(''),
    isIndexing: ref(true),
    /** The core could not be reached: nothing else in the window is true. */
    failure: ref(''),
    /** What the window itself lost touch with, said until it has it back. */
    lost: ref(''),
    error: ref(''),
    unwatched: ref(''),
    /** Why an agent cannot be reached, as the vault last answered. */
    unreachable: ref(''),
    /** Whether the vault holds a note to show at all. */
    hasNote: ref(false),
    /** The note the vault opens with, for whatever has nowhere else to start. */
    opening: ref(''),
    chunks: ref(0),
    embedded: ref(0),
    isEmbedding: ref(false),
    tasks: shallowRef<readonly Task[]>([]),
  }
}

type DisplayState = ReturnType<typeof createDisplayState>

interface StartDisplayDeps {
  state: DisplayState
  ask: () => Promise<unknown>
  readInitialNote: () => Promise<string>
  onVaultChanged: (paths: readonly string[], renames: readonly PathRename[]) => Promise<void> | void
  streams: ReturnType<typeof useWindowStreams>
  wait: (ms: number) => Promise<unknown>
  isOpen: () => boolean
}

async function startDisplay(deps: StartDisplayDeps): Promise<void> {
  const { state, ask, readInitialNote, onVaultChanged, streams, wait, isOpen } = deps
  try {
    while (isOpen()) {
      const vault = (await ask()) as { scan: { error?: string; isReady?: boolean } }
      const note = await readInitialNote()
      if (!isOpen()) return
      if (note) {
        state.isIndexing.value = false
        await onVaultChanged([], [])
        void streams.follow()
        void streams.watch()
        void streams.draw()
        void streams.followTasks()
        return
      }
      if (vault.scan.error || vault.scan.isReady) {
        state.isIndexing.value = false
        void streams.follow()
        void streams.watch()
        void streams.draw()
        void streams.followTasks()
        return
      }
      await wait(100)
    }
  } catch (error) {
    state.failure.value = formatErrorMessage(error)
    state.isIndexing.value = false
  }
}

async function askDisplayState(core: VaultPort, display: DisplayState): Promise<unknown> {
  const [state, agent] = await Promise.all([core.state(), core.agentUnreachable()])
  display.unreachable.value = agent
  display.name.value = state.name
  if (display.at.value === '') display.at.value = state.path
  display.error.value = state.scan.error
  display.unwatched.value = state.scan.unwatchedPath
  display.chunks.value = Number(state.coverage.chunkCount)
  display.embedded.value = Number(state.coverage.embeddedCount)
  display.isEmbedding.value = state.coverage.isEmbedding
  return state
}

async function checkVaultSwapped(core: VaultPort, display: DisplayState): Promise<boolean> {
  const was = display.at.value
  try {
    const state = (await askDisplayState(core, display)) as { path: string }
    return was !== '' && state.path !== was
  } catch {
    return false
  }
}

async function readInitialNotePath(core: VaultPort, display: DisplayState): Promise<string> {
  const note = await core.getInitialOpenPath()
  display.hasNote.value = note !== null
  display.opening.value = note?.path ?? ''
  return display.opening.value
}

export function useWindowDisplay(
  core: VaultPort & Pick<NotePort, 'watchEdits'>,
  how: DisplayOptions = {},
) {
  const wait = how.wait ?? sleep
  const onVaultChanged = how.onVaultChanged ?? (() => {})
  const reload = how.reload ?? (() => {})
  const display = createDisplayState()
  let open = true
  const listening = new AbortController()

  const ask = () => askDisplayState(core, display)
  const readInitialNote = () => readInitialNotePath(core, display)
  const isVaultSwapped = () => checkVaultSwapped(core, display)

  const streams = useWindowStreams({
    core,
    listening,
    isOpen: () => open,
    setLost: (said) => {
      display.lost.value = said
    },
    wait,
    opening: display.opening,
    tasks: display.tasks,
    isVaultSwapped,
    reload,
    onVaultChanged,
    readInitialNote,
    ask,
    reportChange: how.reportChange ?? (() => {}),
    travelTo: how.travelTo ?? (() => {}),
    openFileAt: how.openFileAt ?? (() => {}),
  })

  return {
    ...display,
    readInitialNote,
    start: () =>
      startDisplay({
        state: display,
        ask,
        readInitialNote,
        onVaultChanged,
        streams,
        wait,
        isOpen: () => open,
      }),
    follow: streams.follow,
    watch: streams.watch,
    close: () => {
      open = false
      listening.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
