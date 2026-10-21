/**
 * Which row the tab key reaches, and the keyboard put onto a row once it is drawn.
 */
import { describe, expect, it, vi } from 'vitest'
import { flatten, type Row, type RowId } from '../lib/row'
import { useDrawnRows } from './rows'

const ROWS: readonly Row[] = [
  { id: 'work', name: 'Work', hasChildren: false },
  { id: 'notes', name: 'Notes', hasChildren: false },
  { id: 'loose', name: 'Loose', hasChildren: false },
]

const create = (rows: readonly Row[] = ROWS, selection: readonly RowId[] = []) =>
  useDrawnRows(
    () => flatten(rows, new Set()),
    () => selection,
  )

/** A drawn row: what a row component hands over, with the element under `$el`. */
const createDrawn = () => {
  const focus = vi.fn<() => void>()
  return { focus, drawn: { $el: { focus } } }
}

describe('the row the tab key reaches', () => {
  it('is the first row where nothing else is requested for', () => {
    expect(create().tabbed.value).toBe('work')
  })

  it('is no row while nothing is drawn', () => {
    expect(create([]).tabbed.value).toBeNull()
  })

  it('is the first row of the selection', () => {
    expect(create(ROWS, ['loose', 'notes']).tabbed.value).toBe('loose')
  })

  it('is the row the keyboard was left on', () => {
    const rows = create(ROWS, ['loose'])
    rows.here.value = 'notes'
    expect(rows.tabbed.value).toBe('notes')
  })

  it('is not a row the keyboard was left on once it is no longer drawn', () => {
    const rows = create(ROWS, ['loose'])
    rows.here.value = 'gone'
    expect(rows.tabbed.value).toBe('loose')
  })

  it('is not a selected row that is no longer drawn', () => {
    const rows = create(ROWS, ['gone'])
    expect(rows.tabbed.value).toBe('work')
  })
})

describe('a row as it is drawn', () => {
  it('is found under the row it remains for', () => {
    const rows = create()
    const { drawn } = createDrawn()
    rows.setRowElement('notes', drawn)

    expect(rows.getRowElement('notes')).toBe(drawn.$el)
    expect(rows.getRowElement('work')).toBeNull()
  })

  it('is forgotten once it is taken away', () => {
    const rows = create()
    rows.setRowElement('notes', createDrawn().drawn)
    rows.setRowElement('notes', null)

    expect(rows.getRowElement('notes')).toBeNull()
  })

  it('is forgotten once it is replaced by something that draws nothing', () => {
    const rows = create()
    rows.setRowElement('notes', createDrawn().drawn)
    rows.setRowElement('notes', {})

    expect(rows.getRowElement('notes')).toBeNull()
  })
})

describe('the keyboard put onto a row', () => {
  it('lands on the row once the rows are drawn', async () => {
    const rows = create()
    const { focus, drawn } = createDrawn()
    rows.setRowElement('notes', drawn)

    const isDone = rows.focusRow('notes')
    expect(focus).not.toHaveBeenCalled()
    expect(rows.here.value).toBe('notes')

    await isDone
    expect(focus).toHaveBeenCalledTimes(1)
  })

  it('lands on no row for no row', async () => {
    const rows = create()
    const { focus, drawn } = createDrawn()
    rows.setRowElement('notes', drawn)
    rows.here.value = 'notes'

    await rows.focusRow(null)

    expect(rows.here.value).toBe('notes')
    expect(focus).not.toHaveBeenCalled()
  })

  it('is kept at the row where the row is not drawn', async () => {
    const rows = create()

    await expect(rows.focusRow('gone')).resolves.toBeUndefined()
    expect(rows.here.value).toBe('gone')
  })

  it('calls scroll callback before focusing', async () => {
    const scrollToRow = vi.fn<(index: number) => void>()
    const rows = useDrawnRows(
      () => flatten(ROWS, new Set()),
      () => [],
      scrollToRow,
    )
    const { focus, drawn } = createDrawn()
    rows.setRowElement('notes', drawn)

    await rows.focusRow('notes')

    expect(scrollToRow).toHaveBeenCalledWith(1)
    expect(focus).toHaveBeenCalledTimes(1)
  })

  it('uses explicitly provided scroll callback over default', async () => {
    const defaultScroll = vi.fn<(index: number) => void>()
    const explicitScroll = vi.fn<(index: number) => void>()
    const rows = useDrawnRows(
      () => flatten(ROWS, new Set()),
      () => [],
      defaultScroll,
    )
    const { drawn } = createDrawn()
    rows.setRowElement('notes', drawn)

    await rows.focusRow('notes', explicitScroll)

    expect(defaultScroll).not.toHaveBeenCalled()
    expect(explicitScroll).toHaveBeenCalledWith(1)
  })
})
