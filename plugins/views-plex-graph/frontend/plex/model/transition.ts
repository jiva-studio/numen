/** A frame moving towards each new graph. */
import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import type { PlacedNode } from '../lib/node'
import { arrangePlex, easeOut, interpolatePlex } from '../lib/arrange'
import type { ArrangeInput } from '../lib/arrange'
import type { PlexFrame } from '../lib/frame'
import { browserClock, type Clock } from '@numen/ui'
import type { PlexGraph } from '../../types'

export { browserClock, type Clock }

/** `waiting` holds the nodes of the next arrangement that the frame does not hold yet, while the plex waits for their widgets. */
export interface PlexTransition {
  readonly frame: Ref<PlexFrame>
  readonly waiting: Ref<readonly PlacedNode[]>
  readonly isMoving: Ref<boolean>
}

/**
 * The plex is arranged only when `isReady` holds for the nodes it would show.
 * `isActive` holds once the plex may be arranged at all: a plex whose widgets mount for the waiting nodes is not active before its window is measured.
 */
export interface PlexTransitionOptions {
  readonly duration: () => number
  readonly clock?: Clock | undefined
  readonly isReady?: ((ids: readonly string[]) => boolean) | undefined
  readonly isActive?: (() => boolean) | undefined
}

const EMPTY_FRAME: PlexFrame = {
  nodes: [],
  edges: [],
  extent: { minX: 0, minY: 0, maxX: 0, maxY: 0 },
  overflow: {},
}

/**
 * Hold a frame that moves towards each new graph. One arriving
 * mid-movement re-aims from wherever the plex has got to.
 * The plex is arranged once the widgets of every node it would show are loaded; until then the
 * frame stays as it is, and the first one is empty, so no node is placed by a layout that the widgets change.
 * While it waits, the nodes it would show are chosen once for each new graph or input, and a change in the
 * room of the widgets arranges nothing; once the widgets are loaded it arranges once, and then once for each frame
 * in which the room of the widgets changed.
 */
export function usePlexTransition(
  graph: () => PlexGraph,
  input: () => ArrangeInput | undefined,
  options: PlexTransitionOptions,
): PlexTransition {
  const { duration, clock = browserClock, isReady = () => true, isActive = () => true } = options
  const target = () => arrangePlex(graph(), input())

  const frame = shallowRef<PlexFrame>(EMPTY_FRAME)
  const waiting = shallowRef<readonly PlacedNode[]>([])
  const held = shallowRef<readonly string[]>([])
  const isMoving = ref(false)
  let handle: number | null = null
  let isFramePending = false
  let frameId = 0
  let arrangedRoom: unknown

  const cancelFrame = () => {
    frameId += 1
    isFramePending = false
  }

  const stop = () => {
    if (handle !== null) clock.cancel(handle)
    handle = null
    isMoving.value = false
  }

  const run = (to: PlexFrame) => {
    stop()
    const ms = duration()
    if (ms <= 0 || frame.value.nodes.length === 0) {
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

  const arrangeWhenReady = () => {
    cancelFrame()
    if (!isActive()) return
    arrangedRoom = input()?.room
    const to = target()
    held.value = to.nodes.map((node) => node.id)
    if (!isReady(held.value)) {
      const shown = new Set(frame.value.nodes.map((node) => node.id))
      waiting.value = to.nodes.filter((node) => !shown.has(node.id))
      return
    }
    waiting.value = []
    run(to)
  }

  const arrangeInFrame = () => {
    if (isFramePending || input()?.room === arrangedRoom || !isReady(held.value)) return
    isFramePending = true
    const id = frameId
    clock.schedule(() => {
      if (id === frameId) arrangeWhenReady()
    })
  }

  arrangeWhenReady()
  // One watcher for the graph and the input, so a change of them in one tick arranges once.
  watch([() => graph(), isActive, ...getInputSources(input)], arrangeWhenReady)
  // The widgets settling is what arranges the plex that waited for them.
  watch(
    () => isReady(held.value),
    (isOpen) => {
      if (isOpen) arrangeWhenReady()
    },
  )
  watch(() => input()?.room, arrangeInFrame)
  onScopeDispose(() => {
    cancelFrame()
    stop()
  })

  return { frame, waiting, isMoving }
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

function getInputSources(input: () => ArrangeInput | undefined) {
  return [
    () => input()?.placement,
    () => input()?.options?.viewport?.width,
    () => input()?.options?.viewport?.height,
    () => input()?.options?.margin,
    () => input()?.options?.maxPerCluster,
    () => input()?.options?.minWidth,
    () => input()?.options?.gap,
    () => input()?.options?.lineGap,
    () => input()?.options?.focusGap,
    () => input()?.options?.room,
    () => input()?.options?.focusScale,
    () => input()?.options?.routing?.curvature,
    () => input()?.options?.nodeSize?.width,
    () => input()?.options?.nodeSize?.height,
    () => input()?.options?.focusSize?.width,
    () => input()?.options?.focusSize?.height,
    () => input()?.measure,
    () => input()?.measureLabel,
    () => input()?.labelType,
  ]
}
