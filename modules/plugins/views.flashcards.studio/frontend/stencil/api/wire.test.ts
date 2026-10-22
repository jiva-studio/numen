/**
 * What the window remembers about a stencil file between the vault's answers.
 *
 * What is proved here is the remembering: a file the vault has said nothing
 * last read or write earned, and a file that moved keeps the transcript of it.
 */
import { describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/wire'
import type { Cards, DeckProblem, FieldRenameResult, RenamedField, VaultStencil } from '../../cards'
import { WORDS as words } from '../../cards'
import { ERRORS } from '@numen/wire'
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
  kind: 'fieldNotRenamed',
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
const createWire = (cards: ReturnType<typeof vault>) => {
  const messages: { text: string; how: string | undefined }[] = []
  const wire = createStencilWire(cards as unknown as Cards, (text, how) => {
    messages.push({ text, how })
  })
  return { wire, messages }
}

describe('reading a stencil', () => {
  it('arrives back as the body the tab is dirty against', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(asValue({ stencil: stencil(), at: '12 34 Word.md' }))
    const { wire } = createWire(cards)

    const answer = await wire.read('Word.md')

    expect(answer.ok && answer.value.at).toBe('12 34 Word.md')
    expect(JSON.parse(answer.ok ? answer.value.body : '{}').fields).toEqual(['Front', 'Back'])
    expect(wire.getTitle('Word.md')).toBe('Word')
  })

  it('carries no body at all where the read failed', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(asFailure('notAStencil'))
    const { wire } = createWire(cards)

    expect(await wire.read('Notes.md')).toEqual(asFailure('notAStencil'))
    expect(wire.getProblems('Notes.md')).toEqual([])
    expect(wire.getTitle('Notes.md')).toBeUndefined()
  })

  it('is recorded as no failure of the file where the file moved past the tab', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(asFailure('changed'))
    const { wire } = createWire(cards)

    expect(await wire.read('Notes.md')).toEqual(asFailure('notAStencil'))
  })

  it('leaves a failed read with no file to name in a later rename', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(asFailure('missing'))
    cards.renameField.mockResolvedValue(asValue({ decks: [], cards: 0, notWritten: [], at: '' }))
    const { wire } = createWire(cards)
    await wire.read('Word.md')

    await wire.renameField('Word.md', { from: 'Front', to: 'Face' }, vi.fn())

    expect(cards.renameField).toHaveBeenCalledWith('Word.md', { from: 'Front', to: 'Face' }, null)
  })

  it('holds what is wrong with the file the vault named', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(
      asValue({ stencil: stencil({ problems: [problem] }), at: '' }),
    )
    const { wire } = createWire(cards)

    await wire.read('Word.md')

    expect(wire.getProblems('Word.md')).toEqual([problem])
  })
})

describe('writing a stencil', () => {
  it('sends the fields and the faces the body holds, under the file the tab read', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(asValue({ stencil: stencil(), at: '12 34 Word.md' }))
    cards.writeStencil.mockResolvedValue(asValue({ at: '56 78 Word.md' }))
    const { wire } = createWire(cards)

    const readItem = await wire.read('Word.md')
    const body = readItem.ok ? readItem.value.body : ''
    const answer = await wire.write('Word.md', body, { prose: body, fingerprint: '12 34 Word.md' })

    expect(cards.writeStencil).toHaveBeenCalledWith(
      'Word.md',
      {
        fields: ['Front', 'Back'],
        preamble: '',
        faces: [{ name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' }],
        tail: '',
      },
      '12 34 Word.md',
    )
    expect(answer).toEqual(asValue({ body: '', at: '56 78 Word.md' }))
  })

  it('names no file where the tab read none, and nothing where the body is empty', async () => {
    const cards = vault()
    cards.writeStencil.mockResolvedValue(asValue({ at: '' }))
    const { wire } = createWire(cards)

    await wire.write('Word.md', '')

    expect(cards.writeStencil).toHaveBeenCalledWith(
      'Word.md',
      { fields: [], preamble: '', faces: [], tail: '' },
      null,
    )
  })

  it('keeps the file it last knew where the write failed', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(asValue({ stencil: stencil(), at: '12 34 Word.md' }))
    cards.writeStencil.mockResolvedValue(asFailure('unreadable'))
    const { wire } = createWire(cards)

    await wire.read('Word.md')
    await wire.write('Word.md', '')

    expect(wire.getErrorMessage('Word.md', 'unreadable')).toBe(words.notSaved)
  })
})

describe('what is shown for a fault', () => {
  it('is nothing at all where there is no fault', () => {
    const { wire } = createWire(vault())

    expect(wire.getErrorMessage('Word.md', null)).toBe('')
  })

  it('says the vault was out of reach where neither a read nor a write failed', () => {
    const { wire } = createWire(vault())

    expect(wire.getErrorMessage('Word.md', 'unreadable')).toBe(words.unreachable)
  })

  it('says the note is no stencil where the read failed for that', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(asFailure('notAStencil'))
    const { wire } = createWire(cards)

    await wire.read('Notes.md')

    expect(wire.getErrorMessage('Notes.md', 'notAStencil')).toBe(words.notAStencil)
  })

  it('says the file could not be read where the read failed for anything else', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(asFailure('missing'))
    const { wire } = createWire(cards)

    await wire.read('Notes.md')

    expect(wire.getErrorMessage('Notes.md', 'missing')).toBe(words.notRead)
  })

  it('says the note is no stencil where the write failed for that', async () => {
    const cards = vault()
    cards.writeStencil.mockResolvedValue(asFailure('notAStencil'))
    const { wire } = createWire(cards)

    await wire.write('Notes.md', '')

    expect(wire.getErrorMessage('Notes.md', 'notAStencil')).toBe(words.notAStencil)
  })
})

describe('renaming a field', () => {
  /** What the vault answered a rename with, or the rejection instead. */
  const getRenameAnswer = (over: Partial<RenamedField> = {}): FieldRenameResult =>
    asValue({ decks: [], cards: 0, notWritten: [], at: '', ...over })

  it('asks the vault for nothing where the name is unchanged or empty', async () => {
    const cards = vault()
    const { wire } = createWire(cards)
    const changed = vi.fn()

    await wire.renameField('Word.md', { from: 'Front', to: '' }, changed)
    await wire.renameField('Word.md', { from: 'Front', to: 'Front' }, changed)

    expect(cards.renameField).not.toHaveBeenCalled()
    expect(changed).not.toHaveBeenCalled()
  })

  it('names the file the tab read, and nothing where it read none', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(getRenameAnswer())
    const { wire } = createWire(cards)

    await wire.renameField('Word.md', { from: 'Front', to: 'Face' }, vi.fn())

    expect(cards.renameField).toHaveBeenCalledWith('Word.md', { from: 'Front', to: 'Face' }, null)
  })

  it('counts the cards it reached and names the decks it did not', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(
      getRenameAnswer({
        decks: ['Words.md'],
        cards: 3,
        notWritten: [{ path: 'Old.md', text: '' }],
      }),
    )
    const { wire, messages } = createWire(cards)
    const changed = vi.fn()

    await wire.renameField('Word.md', { from: 'Front', to: 'Face' }, changed)

    expect(messages).toEqual([
      { text: words.renamed(3, 1), how: undefined },
      { text: words.notWritten(['Old.md']), how: 'error' },
    ])
    expect(changed).toHaveBeenCalledWith(['Word.md'])
  })

  it('says nothing where the rename reached no card and missed no deck', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(getRenameAnswer())
    const { wire, messages } = createWire(cards)

    await wire.renameField('Word.md', { from: 'Front', to: 'Face' }, vi.fn())

    expect(messages).toEqual([])
  })

  it('carries the rejection in the words the window shows', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(asFailure('unreadable'))
    const { wire, messages } = createWire(cards)
    const changed = vi.fn()

    await wire.renameField('Word.md', { from: 'Front', to: 'Face' }, changed)

    expect(messages).toEqual([{ text: ERRORS['unreadable'], how: 'error' }])
    expect(changed).not.toHaveBeenCalled()
  })

  it('says the file moved past what the tab read, and reads it again', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(asFailure('changed'))
    const { wire, messages } = createWire(cards)
    const changed = vi.fn()

    await wire.renameField('Word.md', { from: 'Front', to: 'Face' }, changed)

    expect(messages).toEqual([{ text: words.notRenamed, how: 'error' }])
    expect(changed).toHaveBeenCalledWith(['Word.md'])
  })

  it('writes no message at all where the wire was given nowhere to write one', async () => {
    const cards = vault()
    cards.renameField.mockResolvedValue(asFailure('unreadable'))
    const wire = createStencilWire(cards as unknown as Cards)

    await expect(
      wire.renameField('Word.md', { from: 'Front', to: 'Face' }, vi.fn()),
    ).resolves.toBeUndefined()
  })
})

describe('what the window remembers of a file', () => {
  it('is nothing where the vault has returned nothing about it', () => {
    const { wire } = createWire(vault())

    expect(wire.getProblems('Word.md')).toEqual(NOTHING.problems)
    expect(wire.getTitle('Word.md')).toBeUndefined()
  })

  it('is release when the last tab on it closes', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(
      asValue({ stencil: stencil({ problems: [problem] }), at: '' }),
    )
    const { wire } = createWire(cards)
    await wire.read('Word.md')

    wire.forget('Word.md', true)
    expect(wire.getProblems('Word.md')).toEqual([problem])

    wire.forget('Word.md', false)
    expect(wire.getProblems('Word.md')).toEqual([])
    expect(wire.getTitle('Word.md')).toBeUndefined()
  })

  it('follows the file where it moved, and leaves nothing behind', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(
      asValue({ stencil: stencil({ problems: [problem] }), at: '' }),
    )
    const { wire } = createWire(cards)
    await wire.read('Word.md')

    wire.movePaths([
      { from: 'Word.md', to: 'Root.md' },
      { from: 'Gone.md', to: 'Still.md' },
    ])

    expect(wire.getProblems('Root.md')).toEqual([problem])
    expect(wire.getTitle('Root.md')).toBe('Word')
    expect(wire.getProblems('Word.md')).toEqual([])
    expect(wire.getTitle('Still.md')).toBeUndefined()
    expect(wire.getTitle('Word.md')).toBeUndefined()
  })

  it('leaves the transcript of the file moved onto where the one moved from was never read', async () => {
    const cards = vault()
    cards.readStencil.mockResolvedValue(
      asValue({ stencil: stencil({ problems: [problem] }), at: '' }),
    )
    const { wire } = createWire(cards)
    await wire.read('Still.md')

    wire.movePaths([{ from: 'Gone.md', to: 'Still.md' }])

    expect(wire.getProblems('Still.md')).toEqual([problem])
    expect(wire.getTitle('Still.md')).toBe('Word')
  })

  it('takes a title the tab read out of the file itself', () => {
    const { wire } = createWire(vault())

    wire.setTitle('Word.md', 'Root')

    expect(wire.getTitle('Word.md')).toBe('Root')
  })
})
