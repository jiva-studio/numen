/** A frame moving towards each new neighbourhood. */
import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import { arrangePlex, easeOut, interpolatePlex } from '../lib/arrange'
import type { ArrangeInput } from '../lib/arrange'
import type { PlexFrame } from '../lib/frame'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import { browserClock, type Clock } from '@/shared/lib/clock'

export { browserClock, type Clock }

export interface PlexTransition {
  readonly frame: Ref<PlexFrame>
  readonly moving: Ref<boolean>
}

/**
 * Hold a frame that moves towards each new neighbourhood. One arriving
 * mid-movement re-aims from wherever the plex has got to.
 */
export function usePlexTransition(
  neighbourhood: () => PlexNeighbourhood,
  input: () => ArrangeInput | undefined,
  duration: () => number,
  clock: Clock = browserClock,
): PlexTransition {
  const target = () => arrangePlex(neighbourhood(), input())

  const frame = shallowRef<PlexFrame>(target())
  const moving = ref(false)

  let handle: number | null = null

  const stop = () => {
    if (handle !== null) clock.cancel(handle)
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
        handle = clock.schedule(step)
      } else {
        handle = null
        moving.value = false
      }
    }

    handle = clock.schedule(step)
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
