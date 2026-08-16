/** The clock, and the only thing in the plex that knows what time it is. */
import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import { arrangePlex, easeOut, interpolatePlex } from './arrange'
import type { ArrangeInput } from './arrange'
import type { PlexFrame, PlexNeighbourhood } from './model'

/** What the plex needs from the world outside it. */
export interface Environment {
  /** Milliseconds, monotonic. Only differences are used. */
  readonly now: () => number
  readonly schedule: (run: (now: number) => void) => number
  readonly cancel: (handle: number) => void
}

export const browserEnvironment: Environment = {
  now: () => performance.now(),
  schedule: (run) => requestAnimationFrame(run),
  cancel: (handle) => cancelAnimationFrame(handle),
}

export interface PlexTransition {
  readonly frame: Ref<PlexFrame>
  readonly moving: Ref<boolean>
}

/**
 * Hold a frame that moves towards each new neighbourhood rather than jumping.
 * A neighbourhood arriving mid-movement re-aims from wherever the plex is,
 * instead of queueing behind the one in progress.
 */
export function usePlexTransition(
  neighbourhood: () => PlexNeighbourhood,
  input: () => ArrangeInput | undefined,
  duration: () => number,
  environment: Environment = browserEnvironment,
): PlexTransition {
  const target = () => arrangePlex(neighbourhood(), input())

  const frame = shallowRef<PlexFrame>(target())
  const moving = ref(false)

  let handle: number | null = null

  const stop = () => {
    if (handle !== null) environment.cancel(handle)
    handle = null
    moving.value = false
  }

  const run = (to: PlexFrame) => {
    stop()

    const from = frame.value
    const ms = duration()
    if (ms <= 0) {
      frame.value = to
      return
    }

    // Elapsed time comes from the callback, not from a clock read here: a
    // backgrounded tab hands the first callback a stale timestamp and the
    // movement would arrive already finished.
    let started: number | null = null
    moving.value = true

    const step = (now: number) => {
      started ??= now
      const t = Math.min(1, (now - started) / ms)
      frame.value = interpolatePlex(from, to, easeOut(t), input()?.options)
      if (t < 1) {
        handle = environment.schedule(step)
      } else {
        handle = null
        moving.value = false
      }
    }

    handle = environment.schedule(step)
  }

  watch(
    () => neighbourhood(),
    () => run(target()),
  )

  // A change to the arrangement is a move too: the reader still has to be able
  // to follow what became of what.
  watch(() => input(), () => run(target()), { deep: true })

  onScopeDispose(stop)

  return { frame, moving }
}
