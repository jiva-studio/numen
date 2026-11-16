import { onScopeDispose, ref, watch, type Ref } from 'vue'
import { easeOut, lerp } from '../../lib/arrange'
import { browserClock, type Clock } from '../transition'
import { EXPANSION_DURATION } from './expansionDuration'

/**
 * How far open a box is, from fully closed to fully open: closed until the attention has been on it for the wait, then open, and closed again as soon as the attention leaves.
 * `on` names what the attention is on, and is empty when it is on nothing. A change of target restarts the wait. A wait of zero never opens.
 */
export function useExpansion(
  on: () => string | null,
  delay: () => number,
  clock: Clock = browserClock,
): Ref<number> {
  const open = ref(0)
  let expandTimer: ReturnType<typeof setTimeout> | undefined
  let frame: number | null = null

  const stopExpandTimer = () => {
    clearTimeout(expandTimer)
    expandTimer = undefined
  }

  const cancelExpandFrame = () => {
    if (frame !== null) clock.cancel(frame)
    frame = null
  }

  /** The rest of the way, from wherever it has got to, at the speed it opens. */
  const move = (to: number) => {
    cancelExpandFrame()
    const from = open.value
    if (from === to) return

    const span = EXPANSION_DURATION * Math.abs(to - from)
    let started: number | null = null

    // Elapsed time comes from the callback, as it does for a move of the whole
    // picture.
    const step = (now: number) => {
      started ??= now
      const t = Math.min(1, (now - started) / span)
      open.value = lerp(from, to, easeOut(t))
      frame = t < 1 ? clock.schedule(step) : null
    }

    frame = clock.schedule(step)
  }

  watch(
    [on, delay],
    ([what, ms]) => {
      stopExpandTimer()
      move(0)
      if (what === null || ms <= 0) return
      expandTimer = setTimeout(() => move(1), ms)
    },
    { immediate: true },
  )

  onScopeDispose(() => {
    stopExpandTimer()
    cancelExpandFrame()
  })

  return open
}
