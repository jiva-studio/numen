/**
 * What one recording tab holds: the recording being played, its transcript, and
 * the run that writes one down.
 *
 * A transcript grows while a run goes, so the tab is told when work on the
 * recording it holds moves and asks for the words again.
 */
import { computed } from 'vue'
import type { TranscriptState } from './transcript'
import type { Stretch, Task } from '../core'
import type { FileOpeners } from '../tabs/openers'
import type { Kind, WindowHandle } from '../tabs/windowing'
import { RECORDING } from '../tabs/workspace'
import { DROP, PROOFREAD, TRANSCRIBE } from './words'
import RecordingTab from './RecordingTab.vue'

/** What a recording tab asks of the window it is drawn in. */
export interface RecordingTabDeps {
  /**
   * A command asked for over the recording the tab holds, carried out where the
   * commands are. `called` is what the tab calls the recording, which is what a
   * step asking for an answer names.
   */
  runs(id: string, path: string, called: string): void
  /**
   * Whether this build can do a run at all, which decides whether the tab
   * offers it. A window that says nothing offers every run.
   */
  canRun?(run: string): boolean
}

/** What one recording tab holds. */
export type RecordingTabState = ReturnType<typeof transcribed>

/**
 * One recording, with what can be asked about its words: writing them down
 * where there are none, and putting them right or taking them away where there
 * are. None is offered while a run is going, or where this build cannot do it
 * at all.
 */
export function transcribed(read: TranscriptState, asks: RecordingTabDeps) {
  /** Whether this build can do a run. A window that says nothing offers every run. */
  const canRun = (run: string): boolean => asks.canRun?.(run) ?? true

  const written = computed(() => read.times.value.length > 0)

  const transcribable = computed(() => !written.value && !read.working.value && canRun(TRANSCRIBE))
  const proofreadable = computed(() => written.value && !read.working.value && canRun(PROOFREAD))
  const droppable = computed(() => written.value && !read.working.value && canRun(DROP))

  const called = read.path.split('/').pop() ?? read.path
  const transcribes = () => asks.runs(TRANSCRIBE, read.path, called)
  const proofreads = () => asks.runs(PROOFREAD, read.path, called)
  const drops = () => asks.runs(DROP, read.path, called)

  return {
    ...read,
    called,
    transcribable,
    transcribes,
    proofreadable,
    proofreads,
    droppable,
    drops,
  }
}

/**
 * The recording tabs of a window. A recording is its own tab, so the same one
 * opened again is the tab it is already played in.
 */
export function recordingKind(
  handle: WindowHandle,
  opens: (path: string) => TranscriptState,
  asks: RecordingTabDeps,
  puts: FileOpeners,
) {
  const kind: Kind<RecordingTabState> = {
    kind: RECORDING,
    opens: (path) => transcribed(opens(path), asks),
    called: (state) => state.called,
    draws: RecordingTab,
    identity: (path) => path,
    shuts: (state) => {
      state.close()
      return true
    },
    at: (state) => ({ file: state.path, source: 'recording' }),
    attends: (state) => ({
      path: state.path,
      recording: { heard: state.heard.value, length: state.length.value },
    }),
  }

  // The player of recordings. The person is taken to the moment the first of
  // the stretches asked for was spoken at.
  const hears = async (path: string, stretches: readonly Stretch[]) => {
    const id = await handle.opens(RECORDING, path)
    void handle.holds<RecordingTabState>(RECORDING, id)?.reach(...stretches)
  }
  puts.hears((path, stretches) => void hears(path, stretches))

  /**
   * What the application is doing, as it last said. A tab whose recording is
   * named there is being transcribed, and asks for the words again.
   */
  const ticked = (tasks: readonly Task[]) => {
    for (const one of handle.each<RecordingTabState>(RECORDING)) {
      one.state.ticks(tasks.some((task) => task.about === one.state.path))
    }
  }

  /**
   * The transcript of a recording went. Every tab standing on it reads the
   * words again, and finds there are none.
   */
  const dropped = (path: string) => {
    for (const one of handle.each<RecordingTabState>(RECORDING)) {
      if (one.state.path === path) one.state.again()
    }
  }

  return { kind, ticked, dropped }
}
