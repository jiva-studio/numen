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
import type { Position } from '@/shared/lib/geometry'

const ROWS: readonly Row[] = [{ id: 'notes', name: 'Notes', hasChildren: false }]

const create = (
  renamingPath = ref<string | null>(null),
  getMenuAt: (row: string) => Position | null = () => null,
) => {
  const notified: unknown[][] = []
  const tell: TreeGesturesTell = (...args: unknown[]) => {
    notified.push(args)
  }
  const displayed = flatten(ROWS, new Set())
  const rows = useDrawnRows(
    () => displayed,
    () => [],
  )
  const selection = useRowSelection(
    () => displayed,
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
    getShownRows: () => displayed,
    getSelected: () => [],
    renamingPath,
    rows,
    selection,
    drag,
    getMenuAt,
    tell,
  })
  return { gestures, notified: notified }
}

describe('the menu key on a row with no place on screen', () => {
  it('is taken, and asks for no menu', () => {
    const { gestures, notified } = create()
    const event = new KeyboardEvent('keydown', { key: 'ContextMenu', cancelable: true })

    gestures.onKeyDown(event)

    expect(event.defaultPrevented).toBe(true)
    expect(notified).toStrictEqual([])
  })
})

describe('the menu keys on a row with a place on screen', () => {
  const at: Position = { x: 12, y: 34 }

  it('opens the menu on shift and F10', () => {
    const { gestures, notified } = create(ref<string | null>(null), () => at)

    gestures.onKeyDown(new KeyboardEvent('keydown', { key: 'F10', shiftKey: true }))

    expect(notified).toStrictEqual([['menu', 'notes', at]])
  })

  it('leaves F10 alone without shift', () => {
    const { gestures, notified } = create(ref<string | null>(null), () => at)
    const event = new KeyboardEvent('keydown', { key: 'F10', cancelable: true })

    gestures.onKeyDown(event)

    expect(notified).toStrictEqual([])
    expect(event.defaultPrevented).toBe(false)
  })

  it('leaves another key alone when shift is retained', () => {
    const { gestures, notified } = create(ref<string | null>(null), () => at)
    const event = new KeyboardEvent('keydown', { key: 'x', shiftKey: true, cancelable: true })

    gestures.onKeyDown(event)

    expect(notified).toStrictEqual([])
    expect(event.defaultPrevented).toBe(false)
  })
})

describe('a name field losing the keyboard', () => {
  it('stops the renaming', () => {
    const renamingPath = ref<string | null>('notes')
    const { gestures } = create(renamingPath)

    gestures.onFieldBlur()

    expect(renamingPath.value).toBeNull()
  })
})
