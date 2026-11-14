import { nextTick, ref, type Ref } from 'vue'
import { browserClock, measureRect, usePressDrag } from '@numen/ui'
import { moveRow, type RowDraft } from '@/entities/connection'
import { WORDS as words } from '../words'

const DRAG_THRESHOLD = 4

/** The rows being ordered and what an order is written with. */
export interface RowOrderSource {
  readonly drafts: Ref<readonly RowDraft[]>
  readonly getList: () => HTMLElement | null
  readonly write: () => void
  readonly move: (from: number, to: number) => void
}

/**
 * The order of the rows, changed by dragging a grip or by the keyboard: Space picks a row up, the
 * arrows move it, Space drops it and Escape puts it back. A row is written when it is dropped, not
 * while it is held. What happens is announced in a live region.
 */
export function useRowOrder(source: RowOrderSource) {
  const heldKey = ref<string | null>(null)
  const announcement = ref('')
  let heldFrom: readonly RowDraft[] = []

  const findIndexAt = (y: number): number | null => {
    const items = Array.from(source.getList()?.children ?? [])
    if (items.length === 0) return null
    const found = items.findIndex((one) => y < measureRect(one).bottom)
    return found < 0 ? items.length - 1 : found
  }

  const drag = usePressDrag<number, number>({
    getThreshold: () => DRAG_THRESHOLD,
    getClock: () => browserClock,
    getDropTargetAt: (_from, point) => findIndexAt(point.y),
    commit: (from, to) => {
      if (to !== null) source.move(from, to)
    },
  })

  const focusGrip = async (key: string): Promise<void> => {
    await nextTick()
    source.getList()?.querySelector<HTMLElement>(`[data-grip="${key}"]`)?.focus()
  }

  const onGripPress = (index: number, event: PointerEvent): void => {
    if (event.button === 0) drag.startPress(index, event)
  }

  const pickUp = (key: string, index: number): void => {
    heldKey.value = key
    heldFrom = source.drafts.value
    announcement.value = words.connectionRowsPickedUp(index + 1, source.drafts.value.length)
  }

  const drop = (index: number): void => {
    heldKey.value = null
    announcement.value = words.connectionRowsDropped(index + 1, source.drafts.value.length)
    source.write()
  }

  const step = async (key: string, index: number, offset: number): Promise<void> => {
    const target = index + offset
    if (target < 0 || target >= source.drafts.value.length) return
    source.drafts.value = moveRow(source.drafts.value, index, target)
    announcement.value = words.connectionRowsMoved(target + 1, source.drafts.value.length)
    await focusGrip(key)
  }

  const cancel = async (key: string): Promise<void> => {
    source.drafts.value = heldFrom
    heldKey.value = null
    announcement.value = words.connectionRowsCancelled
    await focusGrip(key)
  }

  const onGripKey = async (key: string, event: KeyboardEvent): Promise<void> => {
    const index = source.drafts.value.findIndex((one) => one.key === key)
    const isHeld = heldKey.value === key
    if (event.key === ' ') {
      event.preventDefault()
      if (isHeld) drop(index)
      else pickUp(key, index)
    } else if (isHeld && (event.key === 'ArrowUp' || event.key === 'ArrowDown')) {
      event.preventDefault()
      await step(key, index, event.key === 'ArrowUp' ? -1 : 1)
    } else if (isHeld && event.key === 'Escape') {
      event.preventDefault()
      event.stopPropagation()
      await cancel(key)
    }
  }

  return { heldKey, announcement, drag, onGripPress, onGripKey }
}
