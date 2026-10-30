import { ref, shallowRef } from 'vue'
import type { NotePort } from '@/app/ports/notes'
import type { VaultPort } from '@/app/ports/vault'
import type { NoteEdit } from '@/entities/note'
import type { Task } from '@/shared/notices/task'
import type { PathRename } from '@/shared/paths'
import type { TextRange } from '@/shared/textRange'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import { createWindowStreams } from './useWindowStreams'

/**
 * The vault as the whole window reads it, and the notifications when the vault
 * changes.
 *
 * One window reads the vault once: one stream of changes, one stream of edits,
 * one stream of requested notes, one stream of running tasks, and one set of
 * counts. What is rendered from any of it, and by how many tabs, is the
 * window's concern; nothing here depends on the contents of a tab.
 */
/** What the window passes for reading a vault, alongside the vault itself. */
export interface DisplayOptions {
  wait?(ms: number): Promise<unknown>
  /**
   * Receives vault change notifications, and is awaited. A change carrying no
   * paths is empty: everything showing the vault reads again.
   */
  onVaultChanged?(paths: readonly string[], renamed?: readonly PathRename[]): void | Promise<void>
  /** Receives notifications of a change to a note while it is being created. */
  reportChange?(response: NoteEdit): void
  /** Shows a note to the user, requested from outside the window. */
  travelTo?(path: string): void | Promise<void>
  /**
   * Opens a document at spans of its own text, in the tab it is read in. The
   * first span is brought into view.
   */
  openFileAt?(path: string, spans: readonly TextRange[]): void
  /** What draws the page again, once another vault is under this window. */
  reload?(): void
}

function useDisplayState() {
  const state = {
    name: ref(''),
    at: ref(''),
    isIndexing: ref(true),
    /** The core could not be reached: nothing else in the window is true. */
    failure: ref(''),
    /** What the window itself lost its connection to, reported until restored. */
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
  return state
}

type DisplayState = ReturnType<typeof useDisplayState>

interface StartDisplayDeps {
  state: DisplayState
  ask: () => Promise<unknown>
  readInitialNote: () => Promise<string>
  onVaultChanged: (paths: readonly string[], renames: readonly PathRename[]) => Promise<void> | void
  streams: ReturnType<typeof createWindowStreams>
  wait: (ms: number) => Promise<unknown>
  isOpen: () => boolean
}

function startStreams(streams: ReturnType<typeof createWindowStreams>, state: DisplayState): void {
  const onError = (error: unknown) => {
    state.failure.value = formatErrorMessage(error)
  }
  runInBackground(streams.follow(), onError)
  runInBackground(streams.watch(), onError)
  runInBackground(streams.draw(), onError)
  runInBackground(streams.followTasks(), onError)
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
        startStreams(streams, state)
        return
      }
      if (vault.scan.error || vault.scan.isReady) {
        state.isIndexing.value = false
        startStreams(streams, state)
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
  // No listener is attached by default; each callback does nothing.
  const onVaultChanged = how.onVaultChanged ?? (() => undefined)
  const reload = how.reload ?? (() => undefined)
  const display = useDisplayState()
  let open = true
  const abortController = new AbortController()

  const ask = () => askDisplayState(core, display)
  const readInitialNote = () => readInitialNotePath(core, display)
  const isVaultSwapped = () => checkVaultSwapped(core, display)

  const streams = createWindowStreams({
    core,
    listening: abortController,
    isOpen: () => open,
    setLost: (response) => {
      display.lost.value = response
    },
    wait,
    opening: display.opening,
    tasks: display.tasks,
    isVaultSwapped,
    reload,
    onVaultChanged,
    readInitialNote,
    ask,
    reportChange: how.reportChange ?? (() => undefined),
    travelTo: how.travelTo ?? (() => undefined),
    openFileAt: how.openFileAt ?? (() => undefined),
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
      abortController.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
