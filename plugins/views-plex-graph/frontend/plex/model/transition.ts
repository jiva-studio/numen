/** A frame moving towards each new graph. */
import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import type { PlacedNode } from '../lib/node'
import { arrangePlex, easeOut, interpolatePlex, type NodeRoom } from '../lib/arrange'
import type { ArrangeInput } from '../lib/arrange'
import { isFrameMoved, type PlexFrame } from '../lib/frame'
import { browserClock, type Clock } from '@numen/ui'
import type { PlexGraph } from '../../types'

export { browserClock, type Clock }

/** `waiting` holds the nodes of the next arrangement that the frame does not hold yet, which stay mounted, hidden, until the frame holds them: the nodes whose widgets are measured, and the nodes the frame is about to show. */
export interface PlexTransition {
  readonly frame: Ref<PlexFrame>
  readonly waiting: Ref<readonly PlacedNode[]>
  readonly isMoving: Ref<boolean>
}

/**
 * The plex is arranged only when `isReady` holds for the nodes it would show.
 * `isActive` holds once the plex may be arranged at all: not before the window has reported its size.
 * `hasWidgets` holds when the nodes carry widgets, which are measured before any node is shown.
 */
export interface PlexTransitionOptions {
  readonly getDuration: () => number
  readonly clock?: Clock | undefined
  readonly isReady?: ((ids: readonly string[]) => boolean) | undefined
  readonly isActive?: (() => boolean) | undefined
  readonly hasWidgets?: (() => boolean) | undefined
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
 * For each new graph or input the nodes the window holds without any widget are chosen, and their widgets are mounted,
 * hidden, to be measured. Once they are loaded the plex is arranged once with the room each of them measured, and
 * it moves to that arrangement; the nodes it does not show are unmounted. Until then the frame stays as it is, and the
 * first one is empty, so no node is placed by a layout that the widgets change.
 * A change in the room of the widgets afterwards arranges once for each frame in which it changed, with the same
 * nodes shown, and moves the frame only when a node moved by more than a pixel.
 */
export function usePlexTransition(
  graph: () => PlexGraph,
  input: () => ArrangeInput | undefined,
  options: PlexTransitionOptions,
): PlexTransition {
  const {
    getDuration: duration,
    clock = browserClock,
    isReady = () => true,
    isActive = () => true,
    hasWidgets = () => false,
  } = options
  /** The nodes the window holds without any widget, whose widgets are measured, for the current graph and input. */
  let candidates:
    { readonly nodes: readonly PlacedNode[]; readonly ids: ReadonlySet<string> } | undefined
  /** The nodes the current graph and input are shown with, once they are arranged. */
  let shown: ReadonlySet<string> | undefined
  /** The room the nodes that are not shown had when the shown nodes were chosen: their widgets are gone, and the others are laid out as they were. */
  const leftOut = new Map<string, NodeRoom>()
  let target: PlexFrame | undefined

  const getRoom = (room: ArrangeInput['room']): ArrangeInput['room'] =>
    room && ((node) => leftOut.get(node.id) ?? room(node))

  const getRooms = (room: ArrangeInput['room']) =>
    shown &&
    JSON.stringify(
      graph()
        .nodes.filter((node) => shown?.has(node.id))
        .map((node) => room?.(node)),
    )

  const getCandidates = (given: ArrangeInput) => {
    const { nodes } = arrangePlex(graph(), { ...given, room: undefined })
    return { nodes, ids: new Set(nodes.map((node) => node.id)) }
  }

  const frame = shallowRef<PlexFrame>(EMPTY_FRAME)
  const waiting = shallowRef<readonly PlacedNode[]>([])
  const held = shallowRef<readonly string[]>([])
  const isMoving = ref(false)
  let handle: number | null = null
  let isFramePending = false
  let frameId = 0
  /** The room of the shown nodes when they were arranged, to see whether a new room changes any of them. */
  let arrangedRooms: string | undefined

  const cancelFrame = () => {
    frameId += 1
    isFramePending = false
  }

  const stop = () => {
    if (handle !== null) clock.cancel(handle)
    handle = null
    isMoving.value = false
  }

  /** A node leaves `waiting` once the frame holds it, not before, so its widgets are not mounted twice. */
  const setFrame = (next: PlexFrame) => {
    frame.value = next
    if (waiting.value.length === 0) return
    const held = new Set(next.nodes.map((node) => node.id))
    const rest = waiting.value.filter((node) => !held.has(node.id))
    if (rest.length < waiting.value.length) waiting.value = rest
  }

  const run = (to: PlexFrame) => {
    stop()
    target = to
    const ms = duration()
    if (ms <= 0 || frame.value.nodes.length === 0) {
      setFrame(to)
      return
    }

    isMoving.value = true
    handle = startTransition({
      from: frame.value,
      to,
      ms,
      clock,
      getOptions: () => input()?.options,
      onFrame: setFrame,
      onDone: () => {
        handle = null
        isMoving.value = false
      },
      onNext: (next) => {
        handle = next
      },
    })
  }

  const getNotInFrame = (nodes: readonly PlacedNode[]) => {
    const inFrame = new Set(frame.value.nodes.map((node) => node.id))
    return nodes.filter((node) => !inFrame.has(node.id))
  }

  const holdCandidates = (given: ArrangeInput) => {
    if (hasWidgets()) candidates ??= getCandidates(given)
    held.value = candidates?.nodes.map((node) => node.id) ?? []
    if (isReady(held.value)) return true
    waiting.value = getNotInFrame(candidates?.nodes ?? [])
    return false
  }

  const keepLeftOut = (ids: ReadonlySet<string>, room: ArrangeInput['room']) => {
    if (!room) return
    for (const node of graph().nodes) {
      if (!ids.has(node.id)) leftOut.set(node.id, room(node))
    }
  }

  const arrangeWhenReady = () => {
    cancelFrame()
    if (!isActive()) return
    const given = input() ?? {}
    if (!shown && !holdCandidates(given)) return
    const to = arrangePlex(graph(), {
      ...given,
      room: getRoom(given.room),
      only: candidates?.ids,
      keep: shown,
    })
    const ids = new Set(to.nodes.map((node) => node.id))
    const isStill = shown !== undefined && target !== undefined && !isFrameMoved(target, to)
    if (!shown) keepLeftOut(ids, given.room)
    shown = ids
    arrangedRooms = getRooms(given.room)
    held.value = [...ids]
    waiting.value = getNotInFrame(to.nodes)
    if (!isStill) run(to)
  }

  const arrangeAgain = () => {
    candidates = undefined
    shown = undefined
    arrangedRooms = undefined
    leftOut.clear()
    arrangeWhenReady()
  }

  const arrangeInFrame = () => {
    if (isFramePending || !isReady(held.value) || getRooms(input()?.room) === arrangedRooms) return
    isFramePending = true
    const id = frameId
    clock.schedule(() => {
      if (id === frameId) arrangeWhenReady()
    })
  }

  arrangeAgain()
  // One watcher for the graph and the input, so a change of them in one tick arranges once.
  watch([() => graph(), isActive, ...getInputSources(input)], arrangeAgain)
  // The widgets being loaded is what arranges the plex that waited for them.
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
