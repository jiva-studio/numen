import { computed, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { flatten, type Row } from '../lib/row'
import type { RowDragState } from './drag'
import { createTreeGestures, createTreeRenaming, type TreeGesturesEmit } from './gestures'
import { useDrawnRows } from './rows'
import { useRowSelection } from './selection'
import type { Position } from '@/shared/lib/math/geometry'

const ROWS: readonly Row[] = [{ id: 'pages', name: 'Pages', hasChildren: false }]

const create = (
  renamingPath = ref<string | null>(null),
  getMenuAt: (row: string) => Position | null = () => null,
) => {
  const notifiedCalls: unknown[][] = []
  const emit: TreeGesturesEmit = (...args: unknown[]) => {
    notifiedCalls.push(args)
  }
  const visibleRows = flatten(ROWS, new Set())
  const rows = useDrawnRows(
    () => visibleRows,
    () => [],
  )
  const selection = useRowSelection(
    () => visibleRows,
    () => [],
    vi.fn(),
  )
  const drag: RowDragState = {
    into: computed(() => null),
    before: computed(() => null),
    draggedIds: computed(() => new Set<string>()),
    label: computed(() => null),
    hasMoved: computed(() => false),
    at: computed(() => null),
    startPress: vi.fn(),
  }
  const gestures = createTreeGestures({
    getShownRows: () => visibleRows,
    getSelected: () => [],
    rows,
    selection,
    drag,
    getMenuAt,
    emit,
  })
  const renaming = createTreeRenaming(renamingPath, rows, emit)
  return { gestures, renaming, notified: notifiedCalls, rows, selection }
}

describe('ContextMenu key when getMenuAt returns null', () => {
  it('prevents default and does not emit menu', () => {
    const { gestures, notified } = create()
    const event = new KeyboardEvent('keydown', { key: 'ContextMenu', cancelable: true })

    gestures.onKeyDown(event)

    expect(event.defaultPrevented).toBe(true)
    expect(notified).toStrictEqual([])
  })
})

describe('menu keys when getMenuAt returns a position', () => {
  const pos: Position = { x: 12, y: 34 }

  it('emits menu on Shift+F10', () => {
    const { gestures, notified } = create(ref<string | null>(null), () => pos)

    gestures.onKeyDown(new KeyboardEvent('keydown', { key: 'F10', shiftKey: true }))

    expect(notified).toStrictEqual([['menu', 'pages', pos]])
  })

  it('ignores F10 without Shift', () => {
    const { gestures, notified } = create(ref<string | null>(null), () => pos)
    const event = new KeyboardEvent('keydown', { key: 'F10', cancelable: true })

    gestures.onKeyDown(event)

    expect(notified).toStrictEqual([])
    expect(event.defaultPrevented).toBe(false)
  })

  it('ignores other keys with Shift', () => {
    const { gestures, notified } = create(ref<string | null>(null), () => pos)
    const event = new KeyboardEvent('keydown', { key: 'x', shiftKey: true, cancelable: true })

    gestures.onKeyDown(event)

    expect(notified).toStrictEqual([])
    expect(event.defaultPrevented).toBe(false)
  })
})

describe('renaming handlers', () => {
  it('clears renamingPath on blur, rename, or abandon and restores row focus', async () => {
    const renamingPath = ref<string | null>('pages')
    const { gestures, renaming, notified, rows } = create(renamingPath)
    const focusSpy = vi.spyOn(rows, 'focusRow')

    renaming.onFieldBlur()
    expect(renamingPath.value).toBeNull()

    renamingPath.value = 'pages'
    renaming.onRename('pages', 'Renamed')
    expect(renamingPath.value).toBeNull()
    expect(notified).toStrictEqual([['rename', 'pages', 'Renamed']])
    expect(focusSpy).toHaveBeenCalledWith('pages')

    renamingPath.value = 'pages'
    renaming.onAbandon('pages')
    expect(renamingPath.value).toBeNull()
    expect(focusSpy).toHaveBeenCalledTimes(2)

    gestures.onKeyDown(new KeyboardEvent('keydown', { key: 'ArrowDown' }))
    expect(focusSpy).toHaveBeenCalledTimes(3)
    expect(focusSpy).toHaveBeenLastCalledWith('pages')
  })
})
