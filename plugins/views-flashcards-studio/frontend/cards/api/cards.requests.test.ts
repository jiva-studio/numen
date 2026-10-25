/**
 * What the window sends the cards service for each thing it does with a deck or
 * a stencil, and what it makes of the answer: an error keeps its own words, a
 * file moved on is `changed`, and a missing stamp is the empty string.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/wire'
import { ErrorCode } from '@numen/protocol'
import { cardsService } from './clients'
import { cards } from './cards'

vi.mock('./clients', () => ({
  cardsService: {
    createDeck: vi.fn(),
    readDeck: vi.fn(),
    writeDeck: vi.fn(),
    listStencils: vi.fn(),
    createStencil: vi.fn(),
    renameStencilField: vi.fn(),
    readStencil: vi.fn(),
    writeStencil: vi.fn(),
  },
}))

/** A file the service stamped, and the one string the window carries it as. */
const stampedAt = { path: 'Deck.md', size: 12n, mtime: 34n }
const carried = '12 34 Deck.md'

/** A file the window read once, as the service takes it back. */
const seenAt = { path: 'Deck.md', size: 12n, mtime: 34n }

const deckMessage = {
  path: 'Deck.md',
  title: 'Deck',
  preamble: 'about',
  cards: [],
  sections: [{ name: 'Physics', preamble: 'on motion', extra: 'dropped' }],
  tail: 'after',
  problems: [],
}

const stencilMessage = {
  path: 'Word.md',
  title: 'Word',
  preamble: 'about',
  tail: 'after',
  fields: ['Front', 'Back'],
  faces: [{ name: 'Reading', preamble: 'p', front: '{{Front}}', back: '{{Back}}' }],
  problems: [],
}

/** The service answers once with this, whatever the type of the call. */
const answer = <Name extends keyof typeof cardsService>(name: Name, reply: unknown) =>
  vi.mocked(cardsService[name] as (request: unknown) => Promise<unknown>).mockResolvedValue(reply)

beforeEach(() => {
  vi.clearAllMocks()
})

describe('making a deck', () => {
  it('asks for the title in the folder', async () => {
    answer('createDeck', { path: 'Decks/Words.md' })

    await cards.createDeck('Words', 'Decks')

    expect(cardsService.createDeck).toHaveBeenCalledWith({ title: 'Words', path: 'Decks' })
  })
})

describe('reading a deck', () => {
  it('asks for the path', async () => {
    answer('readDeck', { deck: deckMessage, at: stampedAt })

    await cards.readDeck('Deck.md')

    expect(cardsService.readDeck).toHaveBeenCalledWith({ path: 'Deck.md' })
  })

  it('carries its sections as name and preamble only, and the stamp as one string', async () => {
    answer('readDeck', { deck: deckMessage, at: stampedAt })

    expect(await cards.readDeck('Deck.md')).toEqual(
      asValue({
        deck: {
          path: 'Deck.md',
          title: 'Deck',
          preamble: 'about',
          cards: [],
          sections: [{ name: 'Physics', preamble: 'on motion' }],
          tail: 'after',
          problems: [],
        },
        at: carried,
      }),
    )
  })

  it('carries the empty string where the service stamped nothing', async () => {
    answer('readDeck', { deck: deckMessage })

    const readItem = await cards.readDeck('Deck.md')

    expect(readItem.ok && readItem.value.at).toBe('')
  })

  it('fails in the service words even where a deck came with them', async () => {
    answer('readDeck', { error: ErrorCode.MISSING, deck: deckMessage, bound: 400n })

    expect(await cards.readDeck('Deck.md')).toEqual(asFailure({ code: 'missing', bound: 400 }))
  })

  it('fails in the service words where no deck came', async () => {
    answer('readDeck', { error: ErrorCode.TOO_LARGE, bound: 900n })

    expect(await cards.readDeck('Deck.md')).toEqual(asFailure({ code: 'tooLarge', bound: 900 }))
  })

  it('is no deck where no deck and no error came', async () => {
    answer('readDeck', { bound: 400n })

    expect(await cards.readDeck('Deck.md')).toEqual(asFailure({ code: 'notADeck', bound: 400 }))
  })

  it('is changed where the file moved past what was requested', async () => {
    answer('readDeck', { error: ErrorCode.STALE, bound: 400n })

    expect(await cards.readDeck('Deck.md')).toEqual(asFailure({ code: 'changed', bound: 400 }))
  })
})

describe('writing a deck', () => {
  const deck = {
    preamble: 'about',
    tail: 'after',
    cards: [],
    sections: [{ name: 'Physics', preamble: 'on motion' }],
  }

  it('sends the parts of the deck and the file it was read from', async () => {
    answer('writeDeck', { at: stampedAt })

    await cards.writeDeck('Deck.md', deck, carried)

    expect(cardsService.writeDeck).toHaveBeenCalledWith({
      path: 'Deck.md',
      preamble: 'about',
      cards: [],
      sections: [{ name: 'Physics', preamble: 'on motion' }],
      tail: 'after',
      baseline: seenAt,
    })
  })

  it('answers with the stamp of what it wrote', async () => {
    answer('writeDeck', { at: stampedAt })

    expect(await cards.writeDeck('Deck.md', deck, null)).toEqual(asValue({ at: carried }))
  })

  it('answers with the empty string where the service stamped nothing', async () => {
    answer('writeDeck', {})

    expect(await cards.writeDeck('Deck.md', deck, null)).toEqual(asValue({ at: '' }))
  })

  it('fails in the service words, with the bound', async () => {
    answer('writeDeck', { error: ErrorCode.DECK_TOO_LARGE, bound: 700n })

    expect(await cards.writeDeck('Deck.md', deck, null)).toEqual(
      asFailure({ code: 'deckTooLarge', bound: 700 }),
    )
  })
})

describe('the stencils of a vault', () => {
  it('are requested for up to the limit given', async () => {
    answer('listStencils', { stencils: [], total: 0 })

    await cards.stencils(5)

    expect(cardsService.listStencils).toHaveBeenCalledWith({ limit: 5 })
  })

  it('are requested for up to no limit where none is given', async () => {
    answer('listStencils', { stencils: [], total: 0 })

    await cards.stencils()

    expect(cardsService.listStencils).toHaveBeenCalledWith({ limit: 0 })
  })
})

describe('renaming a field', () => {
  it('sends the path, both names and the file it was read from', async () => {
    answer('renameStencilField', { decks: [], cards: 0, notWritten: [], at: stampedAt })

    await cards.renameField('Word.md', { from: 'Front', to: 'Face' }, carried)

    expect(cardsService.renameStencilField).toHaveBeenCalledWith({
      path: 'Word.md',
      from: 'Front',
      to: 'Face',
      baseline: seenAt,
    })
  })

  it('sends no file where none was read', async () => {
    answer('renameStencilField', { decks: [], cards: 0, notWritten: [] })

    await cards.renameField('Word.md', { from: 'Front', to: 'Face' }, null)

    const dispatched = vi.mocked(cardsService.renameStencilField).mock.calls[0]?.[0]
    expect(dispatched).toEqual({ path: 'Word.md', from: 'Front', to: 'Face' })
    expect(dispatched).not.toHaveProperty('seen')
  })

  it('answers with the empty string where the service stamped nothing', async () => {
    answer('renameStencilField', { decks: [], cards: 0, notWritten: [] })

    const renameResult = await cards.renameField('Word.md', { from: 'Front', to: 'Face' }, null)

    expect(renameResult.ok && renameResult.value.at).toBe('')
  })

  it('fails in the service words', async () => {
    answer('renameStencilField', { error: ErrorCode.MISSING })

    expect(await cards.renameField('Word.md', { from: 'Front', to: 'Face' }, null)).toEqual(
      asFailure('missing'),
    )
  })

  it('is changed where the file moved past what was read', async () => {
    answer('renameStencilField', { error: ErrorCode.STALE })

    expect(await cards.renameField('Word.md', { from: 'Front', to: 'Face' }, carried)).toEqual(
      asFailure('changed'),
    )
  })
})

describe('making a stencil', () => {
  it('asks for the title in the folder with its fields', async () => {
    answer('createStencil', { path: 'Word.md' })

    await cards.createStencil('Word', 'Stencils', ['Front', 'Back'])

    expect(cardsService.createStencil).toHaveBeenCalledWith({
      title: 'Word',
      path: 'Stencils',
      fields: ['Front', 'Back'],
    })
  })
})

describe('reading a stencil', () => {
  it('asks for the path', async () => {
    answer('readStencil', { stencil: stencilMessage, at: stampedAt })

    await cards.readStencil('Word.md')

    expect(cardsService.readStencil).toHaveBeenCalledWith({ path: 'Word.md' })
  })

  it('carries the empty string where the service stamped nothing', async () => {
    answer('readStencil', { stencil: stencilMessage })

    const readItem = await cards.readStencil('Word.md')

    expect(readItem.ok && readItem.value.at).toBe('')
  })

  it('fails in the service words even where a stencil came with them', async () => {
    answer('readStencil', { error: ErrorCode.MISSING, stencil: stencilMessage })

    expect(await cards.readStencil('Word.md')).toEqual(asFailure('missing'))
  })

  it('fails in the service words where no stencil came', async () => {
    answer('readStencil', { error: ErrorCode.UNREADABLE })

    expect(await cards.readStencil('Word.md')).toEqual(asFailure('unreadable'))
  })

  it('is changed where the file moved past what was requested, though a stencil came', async () => {
    answer('readStencil', { error: ErrorCode.STALE, stencil: stencilMessage })

    expect(await cards.readStencil('Word.md')).toEqual(asFailure('changed'))
  })

  it('is no stencil where no stencil and no error came', async () => {
    answer('readStencil', {})

    expect(await cards.readStencil('Word.md')).toEqual(asFailure('notAStencil'))
  })
})

describe('writing a stencil', () => {
  const stencil = {
    fields: ['Front', 'Back'] as const,
    preamble: 'about',
    tail: 'after',
    faces: [{ name: 'Reading', preamble: 'p', front: '{{Front}}', back: '{{Back}}' }],
  }

  it('sends the fields, the faces and the file it was read from', async () => {
    answer('writeStencil', { at: stampedAt })

    await cards.writeStencil('Word.md', stencil, carried)

    expect(cardsService.writeStencil).toHaveBeenCalledWith({
      path: 'Word.md',
      fields: ['Front', 'Back'],
      preamble: 'about',
      faces: [{ name: 'Reading', preamble: 'p', front: '{{Front}}', back: '{{Back}}' }],
      tail: 'after',
      baseline: seenAt,
    })
  })

  it('sends no file where none was read', async () => {
    answer('writeStencil', {})

    await cards.writeStencil('Word.md', { ...stencil, fields: ['Front'] }, null)

    expect(vi.mocked(cardsService.writeStencil).mock.calls[0]?.[0]).not.toHaveProperty('seen')
  })

  it('answers with the empty string where the service stamped nothing', async () => {
    answer('writeStencil', {})

    await cards.writeStencil('Word.md', { ...stencil, fields: ['Front'] }, null)

    expect(await cards.writeStencil('Word.md', { ...stencil, fields: ['Front'] }, null)).toEqual(
      asValue({ at: '' }),
    )
  })

  it('fails in the service words', async () => {
    answer('writeStencil', { error: ErrorCode.OCCUPIED })

    expect(await cards.writeStencil('Word.md', { ...stencil, fields: ['Front'] }, null)).toEqual(
      asFailure('occupied'),
    )
  })

  it('is changed where the file moved past what was read', async () => {
    answer('writeStencil', { error: ErrorCode.STALE })

    expect(await cards.writeStencil('Word.md', { ...stencil, fields: ['Front'] }, carried)).toEqual(
      asFailure('changed'),
    )
  })
})
