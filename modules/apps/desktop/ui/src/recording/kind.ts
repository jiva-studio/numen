/**
 * What one recording tab holds: the recording being played, its transcript, and
 * the run that writes one down.
 *
 * A transcript grows while a run goes, so the tab is told when work on the
 * recording it holds moves and asks for the words again.
 */
import { computed } from 'vue'
import type { Listening } from './listening'
import { canRun } from '../commanding'
import type { Task } from '../core'
import type { Putting } from '../putting'
import type { Host, Kind } from '../windowing'
import { RECORDING } from '../workspace'
import RecordingTab from './RecordingTab.vue'

/** The run a recording tab asks for, under the identity the commands give it. */
export const TRANSCRIBE = 'transcribe'

/** What a recording tab asks of the window it is drawn in. */
export interface Hearing {
  /** A run asked for over the recording the tab holds, carried out where the commands are. */
  runs(id: string, path: string): void
}

/** What one recording tab holds. */
export type Held = ReturnType<typeof transcribed>

/**
 * One recording, with the run that writes its words down. The run is offered
 * where the tab holds no words and none is going, and nowhere this build cannot
 * do it at all.
 */
export function transcribed(listen: Listening, asks: Hearing) {
  const transcribable = computed(
    () => listen.times.value.length === 0 && !listen.working.value && canRun(TRANSCRIBE),
  )

  const transcribes = () => asks.runs(TRANSCRIBE, listen.path)

  return { ...listen, transcribable, transcribes }
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
    called: (held) => held.path.split('/').pop() ?? held.path,
    draws: RecordingTab,
    identity: (path) => path,
    shuts: (held) => {
      held.close()
      return true
    },
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

  return { kind, ticked }
}
