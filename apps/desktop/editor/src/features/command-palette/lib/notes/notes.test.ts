/** How the open files of the window are reached, and where a file opens. */
import { describe, expect, it, vi } from 'vitest'
import { createNotes } from './notes'

const create = () => {
  const noteOpeners = { openFile: vi.fn().mockResolvedValue(undefined) }
  const writeMessage = vi.fn()
  return { noteOpeners, writeMessage, notes: createNotes([], noteOpeners, writeMessage) }
}

describe('opening a file', () => {
  it('opens it in the placement requested where no place in it is given', () => {
    const { notes, noteOpeners } = create()

    notes.openFile('reef/Coral.md', 'Coral', 'split')

    expect(noteOpeners.openFile).toHaveBeenCalledExactlyOnceWith('reef/Coral.md', 'Coral', 'split')
  })

  it('opens it at a line of the note', () => {
    const { notes, noteOpeners } = create()

    notes.openFile('reef/Coral.md', 'Coral', 'current', { line: 7 })

    expect(noteOpeners.openFile).toHaveBeenCalledExactlyOnceWith('reef/Coral.md', 'Coral', {
      where: 'current',
      line: 7,
    })
  })

  it('opens it at the first line where the line is zero', () => {
    const { notes, noteOpeners } = create()

    notes.openFile('reef/Coral.md', 'Coral', 'current', { line: 0 })

    expect(noteOpeners.openFile).toHaveBeenCalledExactlyOnceWith('reef/Coral.md', 'Coral', {
      where: 'current',
      line: 0,
    })
  })

  it('opens a file that is no note at the selector its reader moves to', () => {
    const { notes, noteOpeners } = create()

    notes.openFile('library/a.bin', 'Atlas', 'current', { selector: { page: 3 } })

    expect(noteOpeners.openFile).toHaveBeenCalledExactlyOnceWith('library/a.bin', 'Atlas', {
      where: 'current',
      selector: { page: 3 },
    })
  })

  it('reports a file that could not be opened as an error message', async () => {
    const { notes, noteOpeners, writeMessage } = create()
    noteOpeners.openFile.mockRejectedValue(new Error('the vault is not there'))

    notes.openFile('reef/Coral.md', 'Coral', 'current')
    await vi.waitFor(() => expect(writeMessage).toHaveBeenCalledOnce())

    expect(writeMessage).toHaveBeenCalledWith(expect.any(String), 'error')
  })
})
