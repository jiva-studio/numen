import { afterEach, describe, expect, it, vi } from 'vitest'
import { markdown } from '@codemirror/lang-markdown'
import { ensureSyntaxTree, syntaxTree } from '@codemirror/language'
import { ChangeSet, EditorState, type Extension } from '@codemirror/state'
import { EditorView, WidgetType } from '@codemirror/view'
import { getGrid, isWritable, moveBlock } from './grid'
import { GFM } from '@lezer/markdown'
import { LANGUAGES } from '../../config/languages'
import { createState } from '../../fixtures/state'

const TABLE = '| a | b |\n|---|---|\n| c | d |\n'

const createGrid = (doc: string) => {
  const state = createState(doc)
  const node = syntaxTree(state).topNode.getChild('Table')
  if (!node) throw new Error('no table was parsed')
  return getGrid(state, node)
}

describe('whether the editor takes typing', () => {
  const createStateWith = (...extensions: Extension[]) => EditorState.create({ extensions })

  it('does where it is neither read-only nor closed', () => {
    expect(isWritable(createStateWith())).toBe(true)
  })

  it('does not where it is read-only', () => {
    expect(isWritable(createStateWith(EditorState.readOnly.of(true)))).toBe(false)
  })

  it('does not where it is not editable', () => {
    expect(isWritable(createStateWith(EditorView.editable.of(false)))).toBe(false)
  })

  it('does not where it is both', () => {
    const state = createStateWith(EditorState.readOnly.of(true), EditorView.editable.of(false))
    expect(isWritable(state)).toBe(false)
  })
})

describe('a grid carried by changes', () => {
  it('is the same widget where nothing moved the table', () => {
    const grid = createGrid(TABLE)
    const changes = ChangeSet.of({ from: TABLE.length, insert: 'after' }, TABLE.length)
    expect(moveBlock(grid, changes)).toBe(grid)
  })

  it('remains where the changes carried it', () => {
    const grid = createGrid(TABLE)
    const changes = ChangeSet.of({ from: 0, insert: 'xx' }, TABLE.length)
    const moved = moveBlock(grid, changes) as unknown as typeof grid

    expect(moved).not.toBe(grid)
    expect(moved.table.from).toBe(grid.table.from + 2)
    expect(moved.table.to).toBe(grid.table.to + 2)
    expect(moved.table.head.map((cell) => cell?.from)).toEqual(
      grid.table.head.map((cell) => (cell?.from ?? 0) + 2),
    )
    expect(moved.table.body[0]?.map((cell) => cell?.to)).toEqual(
      grid.table.body[0]?.map((cell) => (cell?.to ?? 0) + 2),
    )
    expect(moved.isWritable).toBe(grid.isWritable)
  })

  it('is any other widget as it was', () => {
    const other = new (class extends WidgetType {
      toDOM() {
        return document.createElement('span')
      }
    })()
    expect(moveBlock(other, ChangeSet.of({ from: 0, insert: 'x' }, 3))).toBe(other)
  })
})

describe('a grid drawn', () => {
  const SOURCE = '| a | b \\| c | d |\n|:--|:-:|--:|\n| e | f |\n| g | h | i |\n'

  const createStateIn = (doc: string, ...extensions: Extension[]) => {
    const state = EditorState.create({
      doc,
      extensions: [markdown({ extensions: GFM, codeLanguages: LANGUAGES }), ...extensions],
    })
    ensureSyntaxTree(state, doc.length, 5000)
    return state
  }

  const createGridIn = (state: EditorState) => {
    const node = syntaxTree(state).topNode.getChild('Table')
    if (!node) throw new Error('no table was parsed')
    return getGrid(state, node)
  }

  const views: EditorView[] = []
  const draw = (grid: ReturnType<typeof getGrid>) => {
    const view = new EditorView({ parent: document.body, state: EditorState.create({}) })
    views.push(view)
    return grid.toDOM(view)
  }

  afterEach(() => {
    views.splice(0).forEach((view) => view.destroy())
    vi.restoreAllMocks()
  })

  const getCells = (frame: HTMLElement, selector: string) =>
    [...frame.querySelectorAll<HTMLElement>(selector)].map((cell) => cell)

  it('is a frame holding the table and the places it remains', () => {
    const state = createStateIn(SOURCE)
    const frame = draw(createGridIn(state))

    expect(frame.tagName).toBe('DIV')
    expect(frame.className).toBe('cm-table')
    expect(frame.dataset['from']).toBe('0')
    expect(frame.dataset['to']).toBe(String(SOURCE.trimEnd().length))
    expect(frame.querySelectorAll('table')).toHaveLength(1)
  })

  it('has the head in a head and the rows in a body, numbered', () => {
    const frame = draw(createGridIn(createStateIn(SOURCE)))

    const head = getCells(frame, 'table > thead > tr')
    expect(head.map((row) => row.dataset['row'])).toEqual(['0'])
    expect(getCells(frame, 'thead th')).toHaveLength(3)
    expect(getCells(frame, 'thead td')).toHaveLength(0)

    const body = getCells(frame, 'table > tbody > tr')
    expect(body.map((row) => row.dataset['row'])).toEqual(['1', '2'])
    expect(getCells(frame, 'tbody td')).toHaveLength(6)
    expect(getCells(frame, 'tbody th')).toHaveLength(0)
  })

  it('numbers the columns and reads what each cell says', () => {
    const frame = draw(createGridIn(createStateIn(SOURCE)))
    const cells = getCells(frame, 'thead .cm-cell')

    expect(cells.map((cell) => cell.dataset['column'])).toEqual(['0', '1', '2'])
    expect(cells.map((cell) => cell.textContent)).toEqual(['a', 'b | c', 'd'])
  })

  it('sets a cell as its column is aligned', () => {
    const frame = draw(createGridIn(createStateIn(SOURCE)))
    const cells = getCells(frame, 'thead .cm-cell')

    expect(cells.map((cell) => cell.style.textAlign)).toEqual(['start', 'center', 'end'])
  })

  it('leaves a cell of an unaligned column unset', () => {
    const frame = draw(createGridIn(createStateIn('| a | b |\n|---|---|\n| c | d |\n')))
    expect(getCells(frame, '.cm-cell').map((cell) => cell.style.textAlign)).toEqual([
      '',
      '',
      '',
      '',
    ])
  })

  it('leaves a cell past the last aligned column unset', () => {
    const frame = draw(createGridIn(createStateIn('| a |\n|:-:|\n| c | d |\n')))
    const cells = getCells(frame, 'tbody .cm-cell')
    expect(cells.map((cell) => cell.style.textAlign)).toEqual(['center', ''])
  })

  it('lets a written cell be typed into, and says where it is written', () => {
    const state = createStateIn(SOURCE)
    const frame = draw(createGridIn(state))
    const cell = getCells(frame, 'thead .cm-cell')[0]

    expect(cell?.getAttribute('contenteditable')).toBe('plaintext-only')
    expect(state.doc.sliceString(Number(cell?.dataset['from']), Number(cell?.dataset['to']))).toBe(
      'a',
    )
  })

  it('leaves a column nothing was written in without a place to type', () => {
    const frame = draw(createGridIn(createStateIn(SOURCE)))
    const cell = getCells(frame, 'tbody tr .cm-cell')[2]

    expect(cell?.hasAttribute('contenteditable')).toBe(false)
    expect(cell?.dataset['from']).toBeUndefined()
    expect(cell?.dataset['to']).toBeUndefined()
    expect(cell?.textContent).toBe('')
  })

  it('has buttons for a column and a row where the editor takes typing', () => {
    const frame = draw(createGridIn(createStateIn(SOURCE)))
    const buttons = getCells(frame, 'button')

    expect(buttons.map((button) => button.className)).toEqual(['cm-add-column', 'cm-add-row'])
    expect(buttons.map((button) => button.textContent)).toEqual(['+', '+'])
    expect(buttons.map((button) => button.tagName)).toEqual(['BUTTON', 'BUTTON'])
  })

  it('is read where the editor is read-only', () => {
    const frame = draw(createGridIn(createStateIn(SOURCE, EditorState.readOnly.of(true))))

    expect(frame.childNodes).toHaveLength(1)
    expect(getCells(frame, 'button')).toHaveLength(0)
    expect(getCells(frame, '.cm-cell').some((cell) => cell.hasAttribute('contenteditable'))).toBe(
      false,
    )
  })

  it('is wired to the editor it is drawn for', () => {
    const state = createStateIn(SOURCE)
    const view = new EditorView({ parent: document.body, state })
    views.push(view)
    const frame = createGridIn(state).toDOM(view)
    document.body.appendChild(frame)

    frame.querySelector<HTMLElement>('.cm-add-row')?.click()

    expect(view.state.doc.lines).toBe(state.doc.lines + 1)
    frame.remove()
  })

  it('is the same grid as one for the same table in the same editor', () => {
    const one = createGridIn(createStateIn(SOURCE))
    expect(one.eq(createGridIn(createStateIn(SOURCE)))).toBe(true)
  })

  it('is not the same grid where the table remains elsewhere', () => {
    const one = createGridIn(createStateIn(SOURCE))
    expect(one.eq(createGridIn(createStateIn(`x\n\n${SOURCE}`)))).toBe(false)
  })

  it('is not the same grid where the table says something else', () => {
    const one = createGridIn(createStateIn(SOURCE))
    expect(one.eq(createGridIn(createStateIn(SOURCE.replace('| e |', '| z |'))))).toBe(false)
  })

  it('is not the same grid where the editor takes typing in one and not in the other', () => {
    const one = createGridIn(createStateIn(SOURCE))
    expect(one.eq(createGridIn(createStateIn(SOURCE, EditorState.readOnly.of(true))))).toBe(false)
  })

  describe('and drawn again', () => {
    it('shows what the table says now, in the frame it has', () => {
      const frame = draw(createGridIn(createStateIn(SOURCE)))
      const before = getCells(frame, '.cm-cell')
      const next = createStateIn(`x\n\n${SOURCE.replace('| e |', '| z |')}`)

      expect(createGridIn(next).updateDOM(frame)).toBe(true)

      const after = getCells(frame, '.cm-cell')
      expect(after).toEqual(before)
      expect(after[3]?.textContent).toBe('z')
      expect(frame.dataset['from']).toBe('3')
      expect(frame.dataset['to']).toBe(String(3 + SOURCE.trimEnd().length))
      expect(
        next.doc.sliceString(Number(after[3]?.dataset['from']), Number(after[3]?.dataset['to'])),
      ).toBe('z')
    })

    it('leaves the text of a cell the person is typing in', () => {
      const frame = draw(createGridIn(createStateIn(SOURCE)))
      const cells = getCells(frame, '.cm-cell')
      vi.spyOn(Document.prototype, 'activeElement', 'get').mockReturnValue(cells[3] ?? null)

      createGridIn(createStateIn(SOURCE.replace('| e |', '| z |'))).updateDOM(frame)

      expect(cells[3]?.textContent).toBe('e')
      expect(cells[4]?.textContent).toBe('f')
    })

    it('puts the places to type where the editor takes typing now', () => {
      const frame = draw(createGridIn(createStateIn(SOURCE)))

      createGridIn(createStateIn(SOURCE, EditorState.readOnly.of(true))).updateDOM(frame)

      expect(getCells(frame, '.cm-cell').some((cell) => cell.hasAttribute('contenteditable'))).toBe(
        false,
      )
      expect(getCells(frame, '.cm-cell').some((cell) => 'from' in cell.dataset)).toBe(false)
      expect(getCells(frame, '.cm-cell').some((cell) => 'to' in cell.dataset)).toBe(false)
    })

    it('leaves a column nothing was written in without a place to type', () => {
      const frame = draw(createGridIn(createStateIn(SOURCE)))

      createGridIn(createStateIn(SOURCE)).updateDOM(frame)

      const cell = getCells(frame, 'tbody tr .cm-cell')[2]
      expect(cell?.hasAttribute('contenteditable')).toBe(false)
      expect(cell?.textContent).toBe('')
    })

    it('is drawn anew where the table has another shape', () => {
      const frame = draw(createGridIn(createStateIn(SOURCE)))
      const next = createStateIn('| a | b |\n|---|---|\n| c | d |\n')

      expect(createGridIn(next).updateDOM(frame)).toBe(true)

      expect(getCells(frame, '.cm-cell').map((cell) => cell.textContent)).toEqual([
        'a',
        'b',
        'c',
        'd',
      ])
      expect(getCells(frame, 'button')).toHaveLength(2)
      expect(frame.dataset['to']).toBe(String('| a | b |\n|---|---|\n| c | d |'.length))
    })
  })
})
