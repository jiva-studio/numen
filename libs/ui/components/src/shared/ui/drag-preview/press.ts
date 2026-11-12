import { onScopeDispose, shallowRef, type ShallowRef } from 'vue'
import type { Position } from '@/shared/lib/math/geometry'
import type { Clock } from '@/shared/lib/time/clock'

interface Drag<Item> {
  readonly item: Item
  readonly hasMoved: boolean
}

/** Callbacks and settings for a press-and-drag gesture. */
export interface Press<Item, At> {
  readonly getThreshold: () => number
  readonly getClock: () => Clock
  readonly getDropTargetAt: (item: Item, point: Position) => At | null
  readonly commit: (item: Item, target: At | null, point: Position) => void
  readonly begin?: (item: Item) => void
  readonly cancel?: (item: Item) => void
}

/** Reactive state and handlers for a press-and-drag gesture. */
export interface PressDragState<Item, At> {
  readonly dragging: ShallowRef<Drag<Item> | null>
  readonly at: ShallowRef<At | null>
  readonly position: ShallowRef<Position | null>
  readonly startPress: (item: Item, event: PointerEvent) => void
}

interface Capture {
  readonly element: Element
  readonly pointer: number
}

/** Tracks a pointer press and starts a drag after the pointer moves past the threshold. */
export function usePressDrag<Item, At>(press: Press<Item, At>): PressDragState<Item, At> {
  const dragging = shallowRef<Drag<Item> | null>(null)
  const dropTarget = shallowRef<At | null>(null)
  const position = shallowRef<Position | null>(null)

  let start: Position
  let capture: Capture | null = null

  const detach = (): void => {
    window.removeEventListener('pointermove', drag)
    window.removeEventListener('pointerup', drop)
    window.removeEventListener('pointercancel', cancel)
    releaseCapture()
  }

  function drag(event: PointerEvent): void {
    const target = dragging.value
    if (!target) return

    const now: Position = { x: event.clientX, y: event.clientY }
    const hasMoved =
      target.hasMoved ||
      Math.abs(now.x - start.x) > press.getThreshold() ||
      Math.abs(now.y - start.y) > press.getThreshold()

    dragging.value = { item: target.item, hasMoved }
    position.value = hasMoved ? now : null
    dropTarget.value = hasMoved ? press.getDropTargetAt(target.item, now) : null

    if (hasMoved && !target.hasMoved) press.begin?.(target.item)
  }

  function drop(event: PointerEvent): void {
    const target = dragging.value
    if (!target) return
    const point: Position = { x: event.clientX, y: event.clientY }

    end()
    if (target.hasMoved) {
      press.commit(target.item, press.getDropTargetAt(target.item, point), point)
    }
  }

  function cancel(): void {
    const target = dragging.value
    if (!target) return

    end()
    if (target.hasMoved) press.cancel?.(target.item)
  }

  function end(): void {
    detach()
    dropTarget.value = null
    position.value = null
    press.getClock().schedule(() => {
      dragging.value = null
    })
  }

  function takeCapture(event: PointerEvent): void {
    const element = event.currentTarget ?? event.target
    if (!(element instanceof Element)) return
    try {
      element.setPointerCapture(event.pointerId)
    } catch {
      return
    }
    capture = { element, pointer: event.pointerId }
  }

  function releaseCapture(): void {
    if (!capture) return
    const { element, pointer } = capture
    capture = null
    if (element.hasPointerCapture(pointer)) element.releasePointerCapture(pointer)
  }

  const startPress = (item: Item, event: PointerEvent): void => {
    start = { x: event.clientX, y: event.clientY }
    dragging.value = { item, hasMoved: false }
    takeCapture(event)
    window.addEventListener('pointermove', drag)
    window.addEventListener('pointerup', drop)
    window.addEventListener('pointercancel', cancel)
  }

  onScopeDispose(() => {
    detach()
    dropTarget.value = null
    position.value = null
    dragging.value = null
  })

  return { dragging, at: dropTarget, position, startPress }
}
