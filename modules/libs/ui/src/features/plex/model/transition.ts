/** A frame moving towards each new neighbourhood. */
import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import { arrangePlex, easeOut, interpolatePlex } from '../lib/arrange'
import type { ArrangeInput } from '../lib/arrange'
import type { PlexFrame } from '../lib/frame'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import { browserClock, type Clock } from '@/shared/lib/time/clock'

export { browserClock, type Clock }

export interface PlexTransition {
  readonly frame: Ref<PlexFrame>
  readonly isMoving: Ref<boolean>
}

export interface PlexTransitionOptions {
  readonly duration: () => number
  readonly clock?: Clock | undefined
}

/**
 * Hold a frame that moves towards each new neighbourhood. One arriving
 * mid-movement re-aims from wherever the plex has got to.
 */
export function usePlexTransition(
  neighbourhood: () => PlexNeighbourhood,
  input: () => ArrangeInput | undefined,
  options: PlexTransitionOptions,
): PlexTransition {
  const { duration, clock = browserClock } = options
  const target = () => arrangePlex(neighbourhood(), input())

  const frame = shallowRef<PlexFrame>(target())
  const isMoving = ref(false)
  let handle: number | null = null

  const stop = () => {
    if (handle !== null) clock.cancel(handle)
    handle = null
    isMoving.value = false
  }

  const run = (to: PlexFrame) => {
    stop()
    const ms = duration()
    if (ms <= 0) {
      frame.value = to
      return
    }

    isMoving.value = true
    handle = startTransition({
      from: frame.value,
      to,
      ms,
      clock,
      getOptions: () => input()?.options,
      onFrame: (f) => {
        frame.value = f
      },
      onDone: () => {
        handle = null
        isMoving.value = false
      },
      onNext: (next) => {
        handle = next
      },
    })
  }

  watch(
    () => neighbourhood(),
    () => run(target()),
  )
  watchInputChanges(input, () => run(target()))
  onScopeDispose(stop)

  return { frame, isMoving }
}

interface StepContext {
  readonly from: PlexFrame
  readonly to: PlexFrame
  readonly ms: number
  readonly clock: Clock
  readonly getOptions: () => ArrangeInput['options'] | undefined
  readonly onFrame: (frame: PlexFrame) => void
  readonly onDone: () => void
  readonly onNext: (handle: number) => void
}

function startTransition(context: StepContext): number {
  let started: number | null = null

  const step = (now: number) => {
    started ??= now
    const t = Math.min(1, (now - started) / context.ms)
    const options = context.getOptions()
    context.onFrame(
      interpolatePlex(
        context.from,
        context.to,
        options ? { progress: easeOut(t), ...options } : easeOut(t),
      ),
    )
    if (t < 1) {
      context.onNext(context.clock.schedule(step))
    } else {
      context.onDone()
    }
  }

  return context.clock.schedule(step)
}

function watchInputChanges(input: () => ArrangeInput | undefined, onChange: () => void): void {
  watch(
    [
      () => input()?.placement,
      () => input()?.options?.viewport?.width,
      () => input()?.options?.viewport?.height,
      () => input()?.options?.margin,
      () => input()?.options?.maxPerLine,
      () => input()?.options?.maxLines,
      () => input()?.options?.minWidth,
      () => input()?.options?.gap,
      () => input()?.options?.lineGap,
      () => input()?.options?.focusGap,
      () => input()?.options?.nodeSize?.width,
      () => input()?.options?.nodeSize?.height,
      () => input()?.options?.focusSize?.width,
      () => input()?.options?.focusSize?.height,
      () => input()?.measure,
      () => input()?.measureLabel,
      () => input()?.labelDepth,
    ],
    onChange,
  )
}
