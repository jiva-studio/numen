/**
 * Which row receives the Tab key, and keyboard focus moving onto a row once it is rendered.
 */
import { describe, expect, it, vi } from 'vitest'
import { flatten, type Row, type RowId } from '../lib/row'
import { useDrawnRows } from './rows'

const ROWS: readonly Row[] = [
  { id: 'work', name: 'Work', hasChildren: false },
  { id: 'pages', name: 'Pages', hasChildren: false },
  { id: 'loose', name: 'Loose', hasChildren: false },
]

const create = (rows: readonly Row[] = ROWS, selection: readonly RowId[] = []) =>
  useDrawnRows(
    () => flatten(rows, new Set()),
    () => selection,
  )

const createDrawnRow = () => {
  const focus = vi.fn<() => void>()
  return { focus, rendered: Object.assign(document.createElement('div'), { $el: { focus } }) }
}

describe('the row the tab key reaches', () => {
  it('is the first row where nothing else is requested for', () => {
    expect(create().tabbed.value).toBe('work')
  })

  it('is no row while nothing is drawn', () => {
    expect(create([]).tabbed.value).toBeNull()
  })

  it('is the first row of the selection', () => {
    expect(create(ROWS, ['loose', 'pages']).tabbed.value).toBe('loose')
  })

  it('is the row the keyboard was left on', () => {
    const rows = create(ROWS, ['loose'])
    rows.here.value = 'pages'
    expect(rows.tabbed.value).toBe('pages')
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
    const { rendered } = createDrawnRow()
    rows.setRowElement('pages', rendered)

    expect(rows.getRowElement('pages')).toBe(rendered.$el)
    expect(rows.getRowElement('work')).toBeNull()
  })

  it('is forgotten once it is taken away', () => {
    const rows = create()
    rows.setRowElement('pages', createDrawnRow().rendered)
    rows.setRowElement('pages', null)

    expect(rows.getRowElement('pages')).toBeNull()
  })

  it('is forgotten once it is replaced by something that draws nothing', () => {
    const rows = create()
    rows.setRowElement('pages', createDrawnRow().rendered)
    rows.setRowElement('pages', document.createElement('div'))

    expect(rows.getRowElement('pages')).toBeNull()
  })
})

describe('the keyboard put onto a row', () => {
  it('lands on the row once the rows are drawn', async () => {
    const rows = create()
    const { focus, rendered } = createDrawnRow()
    rows.setRowElement('pages', rendered)

    const isDone = rows.focusRow('pages')
    expect(focus).not.toHaveBeenCalled()
    expect(rows.here.value).toBe('pages')

    await isDone
    expect(focus).toHaveBeenCalledTimes(1)
  })

  it('lands on no row for no row', async () => {
    const rows = create()
    const { focus, rendered } = createDrawnRow()
    rows.setRowElement('pages', rendered)
    rows.here.value = 'pages'

    await rows.focusRow(null)

    expect(rows.here.value).toBe('pages')
    expect(focus).not.toHaveBeenCalled()
  })

  it('is kept at the row where the row is not drawn', async () => {
    const scrollToRow = vi.fn<(index: number) => void>()
    const rows = useDrawnRows(
      () => flatten(ROWS, new Set()),
      () => [],
      scrollToRow,
    )

    await expect(rows.focusRow('gone')).resolves.toBeUndefined()
    expect(rows.here.value).toBe('gone')
    expect(scrollToRow).not.toHaveBeenCalled()
  })

  it('calls scroll callback before focusing', async () => {
    const scrollToRow = vi.fn<(index: number) => void>()
    const rows = useDrawnRows(
      () => flatten(ROWS, new Set()),
      () => [],
      scrollToRow,
    )
    const { focus, rendered } = createDrawnRow()
    rows.setRowElement('pages', rendered)

    await rows.focusRow('pages')

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
    const { rendered } = createDrawnRow()
    rows.setRowElement('pages', rendered)

    await rows.focusRow('pages', explicitScroll)

    expect(defaultScroll).not.toHaveBeenCalled()
    expect(explicitScroll).toHaveBeenCalledWith(1)
  })
})
