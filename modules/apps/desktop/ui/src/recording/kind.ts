/**
 * What one recording tab holds: the recording being played, its transcript, and
 * the run that writes one down.
 *
 * A transcript grows while a run goes, so the tab is told when work on the
 * recording it holds moves and asks for the words again.
 */
import { computed } from 'vue'
import type { Listening } from './listening'
import type { Task } from '../core'
import type { Putting } from '../putting'
import type { Host, Kind } from '../windowing'
import { RECORDING } from '../workspace'
import RecordingTab from './RecordingTab.vue'

/**
 * What a recording tab asks for, under the identities the commands give them:
 * the words written down, the words put right, and the words taken away.
 */
export const TRANSCRIBE = 'transcribe'
export const PROOFREAD = 'proofread'
export const DROP = 'dropTranscript'

/** What a recording tab asks of the window it is drawn in. */
export interface Hearing {
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
export type Held = ReturnType<typeof transcribed>

/**
 * One recording, with what can be asked about its words: writing them down
 * where there are none, and putting them right or taking them away where there
 * are. None is offered while a run is going, or where this build cannot do it
 * at all.
 */
export function transcribed(listen: Listening, asks: Hearing) {
  /** Whether this build can do a run. A window that says nothing offers every run. */
  const canRun = (run: string): boolean => asks.canRun?.(run) ?? true

  const heard = computed(() => listen.times.value.length > 0)

  const transcribable = computed(() => !heard.value && !listen.working.value && canRun(TRANSCRIBE))
  const proofreadable = computed(() => heard.value && !listen.working.value && canRun(PROOFREAD))
  const droppable = computed(() => heard.value && !listen.working.value && canRun(DROP))

  const called = listen.path.split('/').pop() ?? listen.path
  const transcribes = () => asks.runs(TRANSCRIBE, listen.path, called)
  const proofreads = () => asks.runs(PROOFREAD, listen.path, called)
  const drops = () => asks.runs(DROP, listen.path, called)

  return {
    ...listen,
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
  host: Host,
  opens: (path: string) => Listening,
  asks: Hearing,
  puts: Putting,
) {
  const kind: Kind<Held> = {
    kind: RECORDING,
    opens: (path) => transcribed(opens(path), asks),
    called: (held) => held.called,
    draws: RecordingTab,
    identity: (path) => path,
    shuts: (held) => {
      held.close()
      return true
    },
    at: (held) => ({ file: held.path, source: 'recording' }),
    attends: (held) => ({ path: held.path, at: held.heard.value, of: held.length.value }),
  }

  // The player of recordings. The person is taken to the moment the first of
  // the stretches asked for was spoken at.
  puts.hears(async (path, runs) => {
    const id = await host.opens(RECORDING, path)
    void host.holds<Held>(RECORDING, id)?.reach(...runs)
  })

  /**
   * What the application is doing, as it last said. A tab whose recording is
   * named there is being transcribed, and asks for the words again.
   */
  const ticked = (tasks: readonly Task[]) => {
    for (const one of host.each<Held>(RECORDING)) {
      one.held.ticks(tasks.some((task) => task.about === one.held.path))
    }
  }

  /**
   * The transcript of a recording went. Every tab standing on it reads the
   * words again, and finds there are none.
   */
  const dropped = (path: string) => {
    for (const one of host.each<Held>(RECORDING)) {
      if (one.held.path === path) one.held.again()
    }
  }

  return { kind, ticked, dropped }
}
