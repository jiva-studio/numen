import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { syntaxTree } from '@codemirror/language'
import { EditorView } from '@codemirror/view'
import { createState } from '../fixtures/state'
import { getGrid } from './grid'

const TABLE = '| a | b |\n|---|---|\n| c | d |\n'

const views: EditorView[] = []
const errors: unknown[] = []

const onError = (event: ErrorEvent) => {
  errors.push(event.error)
  event.preventDefault()
}

beforeEach(() => {
  window.addEventListener('error', onError)
})

afterEach(() => {
  window.removeEventListener('error', onError)
  errors.length = 0
  views.splice(0).forEach((view) => view.destroy())
  document.body.replaceChildren()
  vi.restoreAllMocks()
})

const draw = (doc = TABLE) => {
  const state = createState(doc)
  const view = new EditorView({ parent: document.body, state })
  views.push(view)
  const node = syntaxTree(state).topNode.getChild('Table')
  if (!node) throw new Error('no table was parsed')
  const frame = getGrid(state, node).toDOM(view) as HTMLElement
  document.body.appendChild(frame)
  frame
    .querySelectorAll<HTMLElement>('.cm-cell')
    .forEach((cell) => cell.setAttribute('tabindex', '0'))
  return { view, frame }
}

const getCell = (frame: HTMLElement, row: number, column: number) => {
  const found = frame.querySelector<HTMLElement>(
    `tr[data-row="${row}"] .cm-cell[data-column="${column}"]`,
  )
  if (!found) throw new Error('no such cell')
  return found
}

const press = (target: HTMLElement, key: string, shiftKey = false) => {
  const event = new KeyboardEvent('keydown', { key, shiftKey, bubbles: true, cancelable: true })
  target.dispatchEvent(event)
  return event
}

describe('typing in a drawn cell', () => {
  it('writes the cell back over the stretch it came from', () => {
    const { view, frame } = draw()
    const cell = getCell(frame, 1, 1)

    cell.textContent = 'x|y'
    cell.dispatchEvent(new Event('input', { bubbles: true }))

    expect(view.state.doc.toString()).toBe('| a | b |\n|---|---|\n| c | x\\|y |\n')
  })

  it('leaves the document alone where the cell still reads as the text', () => {
    const { view, frame } = draw()
    const dispatch = vi.spyOn(view, 'dispatch')

    getCell(frame, 1, 1).dispatchEvent(new Event('input', { bubbles: true }))

    expect(dispatch).not.toHaveBeenCalled()
  })

  it('leaves the document alone for typing that lands outside a cell', () => {
    const { view, frame } = draw()
    const dispatch = vi.spyOn(view, 'dispatch')

    frame.querySelector('table')?.dispatchEvent(new Event('input', { bubbles: true }))

    expect(dispatch).not.toHaveBeenCalled()
    expect(errors).toEqual([])
  })

  it('leaves the document alone where the cell has no place in it', () => {
    const { view, frame } = draw()
    const dispatch = vi.spyOn(view, 'dispatch')
    const cell = getCell(frame, 1, 1)
    delete cell.dataset['from']

    cell.textContent = 'changed'
    cell.dispatchEvent(new Event('input', { bubbles: true }))

    expect(dispatch).not.toHaveBeenCalled()
  })
})

describe('the buttons that grow the table', () => {
  const getButton = (frame: HTMLElement, name: string) => {
    const found = frame.querySelector<HTMLElement>(`button.${name}`)
    if (!found) throw new Error('no such button')
    return found
  }

  it('keeps a press on a button from moving focus', () => {
    const { frame } = draw()

    const onButton = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    getButton(frame, 'cm-add-row').dispatchEvent(onButton)
    const onCell = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    getCell(frame, 0, 0).dispatchEvent(onCell)

    expect(onButton.defaultPrevented).toBe(true)
    expect(onCell.defaultPrevented).toBe(false)
  })

  it('adds a row under the table', () => {
    const { view, frame } = draw()

    getButton(frame, 'cm-add-row').click()

    expect(view.state.doc.toString()).toBe('| a | b |\n|---|---|\n| c | d |\n| | |\n')
  })

  it('adds a column at the right of the table', () => {
    const { view, frame } = draw()

    getButton(frame, 'cm-add-column').click()

    expect(view.state.doc.toString()).toBe('| a | b | |\n|---|---|---|\n| c | d | |\n')
  })

  it('does nothing for a click that lands on no button', () => {
    const { view, frame } = draw()
    const dispatch = vi.spyOn(view, 'dispatch')

    getCell(frame, 1, 0).click()

    expect(dispatch).not.toHaveBeenCalled()
    expect(errors).toEqual([])
  })

  it('does nothing for a button that adds nothing', () => {
    const { view, frame } = draw()
    const dispatch = vi.spyOn(view, 'dispatch')
    const other = document.createElement('button')
    frame.appendChild(other)

    other.click()

    expect(dispatch).not.toHaveBeenCalled()
    expect(errors).toEqual([])
  })

  it('does nothing where the frame does not say where the table remains', () => {
    const { view, frame } = draw()
    const dispatch = vi.spyOn(view, 'dispatch')
    delete frame.dataset['to']

    getButton(frame, 'cm-add-row').click()
    getButton(frame, 'cm-add-column').click()

    expect(dispatch).not.toHaveBeenCalled()
    expect(errors).toEqual([])
  })

  it('does nothing where the frame does not say where the table starts', () => {
    const { view, frame } = draw()
    const dispatch = vi.spyOn(view, 'dispatch')
    delete frame.dataset['from']

    getButton(frame, 'cm-add-column').click()

    expect(dispatch).not.toHaveBeenCalled()
    expect(errors).toEqual([])
  })
})

describe('pasting into a cell', () => {
  const paste = (target: HTMLElement, text: string | null) => {
    const event = new Event('paste', { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'clipboardData', {
      value:
        text === null ? null : { getData: (type: string) => (type === 'text/plain' ? text : '') },
    })
    target.dispatchEvent(event)
    return event
  }

  it('inserts the text as one line of cell text', () => {
    const { frame } = draw()
    const execCommand = vi.fn<(command: string, showUi?: boolean, value?: string) => boolean>(
      () => true,
    )
    document.execCommand = execCommand

    const event = paste(getCell(frame, 1, 0), 'one\ntwo | three')

    expect(event.defaultPrevented).toBe(true)
    expect(execCommand).toHaveBeenCalledWith('insertText', false, 'one two \\| three')
  })

  it('leaves a paste with no clipboard to the browser', () => {
    const { frame } = draw()
    const execCommand = vi.fn<(command: string, showUi?: boolean, value?: string) => boolean>(
      () => true,
    )
    document.execCommand = execCommand

    const event = paste(getCell(frame, 1, 0), null)

    expect(event.defaultPrevented).toBe(false)
    expect(execCommand).not.toHaveBeenCalled()
  })
})

describe('the keys in a cell', () => {
  it('ignores a key pressed outside any cell', () => {
    const { frame } = draw()

    const event = press(frame.querySelector('table') as HTMLElement, 'Tab')

    expect(event.defaultPrevented).toBe(false)
    expect(errors).toEqual([])
  })

  it('leaves a key it has no use for to the browser', () => {
    const { frame } = draw()

    const event = press(getCell(frame, 1, 0), 'a')

    expect(event.defaultPrevented).toBe(false)
  })

  describe('Escape', () => {
    it('puts the caret at the end of the table line and gives the editor focus', () => {
      const { view, frame } = draw()
      const focus = vi.spyOn(view, 'focus')

      const event = press(getCell(frame, 1, 0), 'Escape')

      expect(event.defaultPrevented).toBe(true)
      expect(view.state.selection.main.head).toBe('| a | b |\n|---|---|\n| c | d |'.length)
      expect(focus).toHaveBeenCalledTimes(1)
    })

    it('only gives the editor focus where the cell has no place in the document', () => {
      const { view, frame } = draw()
      const focus = vi.spyOn(view, 'focus')
      const dispatch = vi.spyOn(view, 'dispatch')
      const cell = getCell(frame, 1, 0)
      delete cell.dataset['to']

      press(cell, 'Escape')

      expect(dispatch).not.toHaveBeenCalled()
      expect(focus).toHaveBeenCalledTimes(1)
      expect(errors).toEqual([])
    })
  })

  describe('Enter', () => {
    it('moves to the cell below', () => {
      const { frame } = draw()

      const event = press(getCell(frame, 0, 1), 'Enter')

      expect(event.defaultPrevented).toBe(true)
      expect(document.activeElement).toBe(getCell(frame, 1, 1))
    })

    it('stays where it is in the last row', () => {
      const { frame } = draw()
      const last = getCell(frame, 1, 0)
      last.focus()

      const event = press(last, 'Enter')

      expect(event.defaultPrevented).toBe(true)
      expect(document.activeElement).toBe(last)
    })
  })

  describe('Tab', () => {
    it('moves to the next cell, across the row end', () => {
      const { frame } = draw()

      const event = press(getCell(frame, 0, 1), 'Tab')

      expect(event.defaultPrevented).toBe(true)
      expect(document.activeElement).toBe(getCell(frame, 1, 0))
    })

    it('puts the caret after the text of the cell it moves to', () => {
      const { frame } = draw()

      press(getCell(frame, 0, 0), 'Tab')

      const selection = window.getSelection()
      const target = getCell(frame, 0, 1)
      expect(selection?.rangeCount).toBe(1)
      expect(selection?.anchorNode).toBe(target)
      expect(selection?.anchorOffset).toBe(1)
      expect(selection?.isCollapsed).toBe(true)
    })

    it('moves back with Shift', () => {
      const { frame } = draw()

      press(getCell(frame, 1, 0), 'Tab', true)

      expect(document.activeElement).toBe(getCell(frame, 0, 1))
    })

    it('stays on the first cell with Shift', () => {
      const { frame } = draw()
      const first = getCell(frame, 0, 0)

      press(first, 'Tab', true)

      expect(document.activeElement).toBe(first)
    })

    it('writes a row from the last cell', () => {
      const { view, frame } = draw()

      const event = press(getCell(frame, 1, 1), 'Tab')

      expect(event.defaultPrevented).toBe(true)
      expect(view.state.doc.toString()).toBe('| a | b |\n|---|---|\n| c | d |\n| | |\n')
    })

    it('writes nothing from the last cell where it has no place in the document', () => {
      const { view, frame } = draw()
      const dispatch = vi.spyOn(view, 'dispatch')
      const last = getCell(frame, 1, 1)
      delete last.dataset['to']

      press(last, 'Tab')

      expect(dispatch).not.toHaveBeenCalled()
      expect(errors).toEqual([])
    })
  })
})
