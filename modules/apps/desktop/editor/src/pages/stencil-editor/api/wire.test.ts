/**
 * What the window remembers about a stencil file between the vault's answers.
 *
 * What is proved here is the remembering: a file the vault has said nothing
 * about yet stands at nothing, the message shown for a fault is the one the
 * last read or write earned, and a file that moved keeps what was said of it.
 */
import { describe, expect, it, vi } from 'vitest'
import type { Cards, DeckProblem, VaultStencil } from '@/entities/deck'
import { WORDS as words } from '@/entities/deck'
import { ERRORS } from '@/shared/words'
import type { ErrorCode } from '@/shared/errors'
import { createStencilWire, NOTHING } from './wire'

/** One stencil as the vault reads it, with whatever a test wants said of it. */
const stencil = (over: Partial<VaultStencil> = {}): VaultStencil => ({
  path: 'Word.md',
  title: 'Word',
  fields: ['Front', 'Back'],
  preamble: '',
  faces: [{ name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' }],
  tail: '',
  problems: [],
  ...over,
})

/** One thing wrong with a stencil, in the shape the vault names it. */
const problem: DeckProblem = {
  fault: 'fieldNotRenamed',
  card: null,
  face: null,
  field: 'Front',
  text: 'Front',
}

/** A vault answering whatever a test tells it to, and nothing else. */
const vault = () => ({
  stencils: vi.fn(),
  createDeck: vi.fn(),
  createStencil: vi.fn(),
  renameField: vi.fn(),
  readDeck: vi.fn(),
  writeDeck: vi.fn(),
  readStencil: vi.fn(),
  writeStencil: vi.fn(),
})

/** The wire over that vault, and the messages it wrote. */
const wireOver = (cards: ReturnType<typeof vault>) => {
  const said: { text: string; how: string | undefined }[] = []
  const wire = createStencilWire(cards as unknown as Cards, (text, how) => {
    said.push({ text, how })
  })
  return { wire, said }
}

describe('reading a stencil', () => {
  it('comes back as the body the tab is dirty against', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({ stencil: stencil(), error: null, at: '12 34 Word.md' })
    const { wire } = wireOver(cards)

    const answer = await wire.read('Word.md')

    expect(answer.error).toBeNull()
    expect(answer.at).toBe('12 34 Word.md')
    expect(JSON.parse(answer.body).fields).toEqual(['Front', 'Back'])
    expect(wire.getTitle('Word.md')).toBe('Word')
  })

  it('carries no body at all where the read was refused', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({ stencil: null, error: 'notAStencil', at: '' })
    const { wire } = wireOver(cards)

    expect(await wire.read('Notes.md')).toEqual({ body: '', error: 'notAStencil' })
    expect(wire.getProblems('Notes.md')).toEqual([])
    expect(wire.getTitle('Notes.md')).toBeUndefined()
  })

  it('holds what is wrong with the file the vault named', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({
      stencil: stencil({ problems: [problem] }),
      error: null,
      at: '',
    })
    const { wire } = wireOver(cards)

    await wire.read('Word.md')

    expect(wire.getProblems('Word.md')).toEqual([problem])
  })
})

describe('writing a stencil', () => {
  it('sends the fields and the faces the body holds, under the file the tab read', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({ stencil: stencil(), error: null, at: '12 34 Word.md' })
    cards.writeStencil.mockResolvedValue({ error: null, changed: false, at: '56 78 Word.md' })
    const { wire } = wireOver(cards)

    const { body } = await wire.read('Word.md')
    const answer = await wire.write('Word.md', body, { prose: body, at: '12 34 Word.md' })

    expect(cards.writeStencil).toHaveBeenCalledWith(
      'Word.md',
      ['Front', 'Back'],
      {
        preamble: '',
        faces: [{ name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' }],
        tail: '',
      },
      '12 34 Word.md',
    )
    expect(answer).toEqual({ body: '', changed: false, error: null, at: '56 78 Word.md' })
  })

  it('names no file where the tab read none, and nothing where the body is empty', async () => {
    const cards = vault()
    cards.writeStencil.mockResolvedValue({ error: null, changed: false, at: '' })
    const { wire } = wireOver(cards)

    await wire.write('Word.md', '')

    expect(cards.writeStencil).toHaveBeenCalledWith(
      'Word.md',
      [],
      { preamble: '', faces: [], tail: '' },
      null,
    )
  })

  it('keeps the file it last knew where the write was refused', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({ stencil: stencil(), error: null, at: '12 34 Word.md' })
    cards.writeStencil.mockResolvedValue({ error: 'unreadable', changed: false, at: '' })
    const { wire } = wireOver(cards)

    await wire.read('Word.md')
    await wire.write('Word.md', '')

    expect(wire.getErrorMessage('Word.md', 'unreadable')).toBe(words.notSaved)
  })
})

describe('what is shown for a fault', () => {
  it('is nothing at all where there is no fault', () => {
    const { wire } = wireOver(vault())

    expect(wire.getErrorMessage('Word.md', null)).toBe('')
  })

  it('says the vault was out of reach where neither a read nor a write failed', () => {
    const { wire } = wireOver(vault())

    expect(wire.getErrorMessage('Word.md', 'unreadable')).toBe(words.unreachable)
  })

  it('says the note is no stencil where the read was refused for that', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({ stencil: null, error: 'notAStencil', at: '' })
    const { wire } = wireOver(cards)

    await wire.read('Notes.md')

    expect(wire.getErrorMessage('Notes.md', 'notAStencil')).toBe(words.notAStencil)
  })

  it('says the file could not be read where the read was refused for anything else', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({ stencil: null, error: 'missing', at: '' })
    const { wire } = wireOver(cards)

    await wire.read('Notes.md')

    expect(wire.getErrorMessage('Notes.md', 'missing')).toBe(words.notRead)
  })

  it('says the note is no stencil where the write was refused for that', async () => {
    const cards = vault()
    cards.writeStencil.mockResolvedValue({ error: 'notAStencil', changed: false, at: '' })
    const { wire } = wireOver(cards)

    await wire.write('Notes.md', '')

    expect(wire.getErrorMessage('Notes.md', 'notAStencil')).toBe(words.notAStencil)
  })
})

describe('renaming a field', () => {
  const getRenameAnswer = (over: Record<string, unknown> = {}) => ({
    decks: [],
    cards: 0,
    notWritten: [],
    error: null as ErrorCode | null,
    changed: false,
    at: '',
    ...over,
  })

  it('asks the vault for nothing where the name is unchanged or empty', async () => {
    const cards = vault()
    const { wire } = wireOver(cards)
    const changed = vi.fn()

    await wire.renameField('Word.md', 'Front', '', changed)
    await wire.renameField('Word.md', 'Front', 'Front', changed)

    expect(cards.renameField).not.toHaveBeenCalled()
    expect(changed).not.toHaveBeenCalled()
  })

  it('names the file the tab read, and nothing where it read none', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(getRenameAnswer())
    const { wire } = wireOver(cards)

    await wire.renameField('Word.md', 'Front', 'Face', vi.fn())

    expect(cards.renameField).toHaveBeenCalledWith('Word.md', 'Front', 'Face', null)
  })

  it('counts the cards it reached and names the decks it did not', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(
      getRenameAnswer({ decks: ['Words.md'], cards: 3, notWritten: [{ path: 'Old.md', text: '' }] }),
    )
    const { wire, said } = wireOver(cards)
    const changed = vi.fn()

    await wire.renameField('Word.md', 'Front', 'Face', changed)

    expect(said).toEqual([
      { text: words.renamed(3, 1), how: undefined },
      { text: words.notWritten(['Old.md']), how: 'error' },
    ])
    expect(changed).toHaveBeenCalledWith(['Word.md'])
  })

  it('says nothing where the rename reached no card and missed no deck', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(getRenameAnswer())
    const { wire, said } = wireOver(cards)

    await wire.renameField('Word.md', 'Front', 'Face', vi.fn())

    expect(said).toEqual([])
  })

  it('carries the refusal in the words the window shows', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(getRenameAnswer({ error: 'unreadable' }))
    const { wire, said } = wireOver(cards)
    const changed = vi.fn()

    await wire.renameField('Word.md', 'Front', 'Face', changed)

    expect(said).toEqual([{ text: ERRORS['unreadable'], how: 'error' }])
    expect(changed).not.toHaveBeenCalled()
  })

  it('says the file moved past what the tab read, and reads it again', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(getRenameAnswer({ changed: true }))
    const { wire, said } = wireOver(cards)
    const changed = vi.fn()

    await wire.renameField('Word.md', 'Front', 'Face', changed)

    expect(said).toEqual([{ text: words.notRenamed, how: 'error' }])
    expect(changed).toHaveBeenCalledWith(['Word.md'])
  })

  it('writes no message at all where the wire was given nowhere to write one', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(getRenameAnswer({ error: 'unreadable' }))
    const wire = createStencilWire(cards as unknown as Cards)

    await expect(wire.renameField('Word.md', 'Front', 'Face', vi.fn())).resolves.toBeUndefined()
  })
})

describe('what the window remembers of a file', () => {
  it('is nothing where the vault has said nothing about it', () => {
    const { wire } = wireOver(vault())

    expect(wire.getProblems('Word.md')).toEqual(NOTHING.problems)
    expect(wire.getTitle('Word.md')).toBeUndefined()
  })

  it('is let go of when the last tab on it closes', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({
      stencil: stencil({ problems: [problem] }),
      error: null,
      at: '',
    })
    const { wire } = wireOver(cards)
    await wire.read('Word.md')

    wire.forget('Word.md', true)
    expect(wire.getProblems('Word.md')).toEqual([problem])

    wire.forget('Word.md', false)
    expect(wire.getProblems('Word.md')).toEqual([])
    expect(wire.getTitle('Word.md')).toBeUndefined()
  })

  it('follows the file where it moved, and leaves nothing behind', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue({
      stencil: stencil({ problems: [problem] }),
      error: null,
      at: '',
    })
    const { wire } = wireOver(cards)
    await wire.read('Word.md')

    wire.movePaths([
      { from: 'Word.md', to: 'Root.md' },
      { from: 'Gone.md', to: 'Still.md' },
    ])

    expect(wire.getProblems('Root.md')).toEqual([problem])
    expect(wire.getTitle('Root.md')).toBe('Word')
    expect(wire.getProblems('Word.md')).toEqual([])
    expect(wire.getTitle('Still.md')).toBeUndefined()
  })

  it('takes a title the tab read out of the file itself', () => {
    const { wire } = wireOver(vault())

    wire.setTitle('Word.md', 'Root')

    expect(wire.getTitle('Word.md')).toBe('Root')
  })
})
