/**
 * What the keyboard asks of a row whose place on screen is not known.
 */
import { computed, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { flatten, type Row } from '../lib/row'
import type { RowDragState } from './drag'
import { useTreeGestures, type TreeGesturesTell } from './gestures'
import { useDrawnRows } from './rows'
import { useRowSelection } from './selection'

const ROWS: readonly Row[] = [{ id: 'notes', name: 'Notes', hasChildren: false }]

const create = () => {
  const told: unknown[][] = []
  const tell: TreeGesturesTell = (...args: unknown[]) => {
    told.push(args)
  }
  const shown = flatten(ROWS, new Set())
  const rows = useDrawnRows(
    () => shown,
    () => [],
  )
  const selection = useRowSelection(
    () => shown,
    () => [],
    () => {},
  )
  const drag: RowDragState = {
    into: computed(() => null),
    before: computed(() => null),
    lifted: computed(() => new Set<string>()),
    label: computed(() => null),
    hasMoved: computed(() => false),
    at: computed(() => null),
    lift: () => {},
  }
  const gestures = useTreeGestures({
    getShownRows: () => shown,
    getSelected: () => [],
    renamingPath: ref<string | null>(null),
    rows,
    selection,
    drag,
    getMenuAt: () => null,
    tell,
  })
  return { gestures, told }
}

describe('the menu key on a row with no place on screen', () => {
  it('is taken, and asks for no menu', () => {
    const { gestures, told } = create()
    const event = new KeyboardEvent('keydown', { key: 'ContextMenu', cancelable: true })

    gestures.onKeyDown(event)

    expect(event.defaultPrevented).toBe(true)
    expect(told).toStrictEqual([])
  })
})
