/**
 * What one recording tab holds: the recording being played, and the words heard
 * in it.
 *
 * A transcript grows while a model listens, so the tab is told when work on the
 * recording it holds moves and asks for the words again.
 */
import type { Listening } from './listening'
import type { Task } from '../core'
import type { Putting } from '../putting'
import type { Host, Kind } from '../windowing'
import { RECORDING } from '../workspace'
import RecordingTab from './RecordingTab.vue'

/** What one recording tab holds. */
export type Held = Listening

/**
 * The recording tabs of a window. A recording is its own tab, so the same one
 * opened again is the tab it is already played in.
 */
export function recordingKind(host: Host, opens: (path: string) => Held, puts: Putting) {
  const kind: Kind<Held> = {
    kind: RECORDING,
    opens,
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
   * named there is being listened to, and asks for the words again.
   */
  const ticked = (tasks: readonly Task[]) => {
    for (const one of host.each<Held>(RECORDING)) {
      one.held.ticks(tasks.some((task) => task.about === one.held.path))
    }
  }

  return { kind, ticked }
}
