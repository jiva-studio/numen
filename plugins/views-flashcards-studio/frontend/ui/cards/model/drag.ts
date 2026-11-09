import { shallowRef, type ShallowRef } from 'vue'
import { getStepInsertionPoint, type InsertionPoint, type StepDirection } from '../../../lib/cards'

/** Dependencies for ordering and moving dragged items. */
export interface DragDeps<At extends InsertionPoint | undefined> {
  readonly order: () => readonly string[]
  readonly nowhere: At
  readonly isMoved: (dragged: string, target: InsertionPoint) => boolean
  readonly move: (dragged: string, target: InsertionPoint) => void
}

/** Reactive state and handlers for reordering items by drag or keyboard. */
export interface DragState<At extends InsertionPoint | undefined> {
  readonly dragged: ShallowRef<string | null>
  readonly at: ShallowRef<At>
  readonly startDrag: (what: string, press: DragEvent) => void
  readonly hover: (target: At, press: DragEvent) => void
  readonly release: () => void
  readonly drop: () => void
  readonly step: (what: string, direction: StepDirection, press: KeyboardEvent) => void
}

/** Tracks drag-and-drop and keyboard reordering state for a list of items. */
export function useDrag<At extends InsertionPoint | undefined>(drag: DragDeps<At>): DragState<At> {
  const dragged = shallowRef<string | null>(null)
  const dropTarget: ShallowRef<At> = shallowRef(drag.nowhere)

  const startDrag = (what: string, press: DragEvent): void => {
    dragged.value = what
    press.dataTransfer?.setData('text/plain', what)
  }

  const hover = (lands: At, press: DragEvent): void => {
    if (dragged.value === null) return
    press.preventDefault()
    dropTarget.value = lands
  }

  const release = (): void => {
    dragged.value = null
    dropTarget.value = drag.nowhere
  }

  const drop = (): void => {
    const target = dragged.value
    const lands = dropTarget.value
    release()
    if (target === null || lands === undefined) return
    if (drag.isMoved(target, lands)) drag.move(target, lands)
  }

  const step = (what: string, direction: StepDirection, press: KeyboardEvent): void => {
    const lands = getStepInsertionPoint(drag.order(), what, direction)
    if (lands === undefined || !drag.isMoved(what, lands)) return
    press.preventDefault()
    drag.move(what, lands)
  }

  return { dragged, at: dropTarget, startDrag, hover, release, drop, step }
}
