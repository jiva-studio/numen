/**
 * What the window sends the cards service for each thing it does with a deck or
 * a card model, and what it makes of the answer: an error keeps its own words, a
 * file moved on is `changed`, and a missing stamp is the empty string.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/plugin-sdk'
import { ErrorCode } from '../../gen/numen/flashcards/v1/errors_pb.js'
import { cardsService } from './clients'
import { cards } from './cards'

vi.mock('./clients', () => ({
  cardsService: {
    createDeck: vi.fn(),
    readDeck: vi.fn(),
    writeDeck: vi.fn(),
    listCardModels: vi.fn(),
    createCardModel: vi.fn(),
    renameCardModelField: vi.fn(),
    readCardModel: vi.fn(),
    writeCardModel: vi.fn(),
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

const cardModelMessage = {
  path: 'Word.md',
  title: 'Word',
  preamble: 'about',
  tail: 'after',
  fields: ['Front', 'Back'],
  templates: [{ name: 'Reading', preamble: 'p', front: '{{Front}}', back: '{{Back}}' }],
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

describe('the card models of a vault', () => {
  it('are requested for up to the limit given', async () => {
    answer('listCardModels', { cardModels: [], total: 0 })

    await cards.cardModels(5)

    expect(cardsService.listCardModels).toHaveBeenCalledWith({ limit: 5 })
  })

  it('are requested for up to no limit where none is given', async () => {
    answer('listCardModels', { cardModels: [], total: 0 })

    await cards.cardModels()

    expect(cardsService.listCardModels).toHaveBeenCalledWith({ limit: 0 })
  })
})

describe('renaming a field', () => {
  it('sends the path, both names and the file it was read from', async () => {
    answer('renameCardModelField', { decks: [], cards: 0, notWritten: [], at: stampedAt })

    await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, carried)

    expect(cardsService.renameCardModelField).toHaveBeenCalledWith({
      path: 'Word.md',
      from: 'Front',
      to: 'Template',
      baseline: seenAt,
    })
  })

  it('sends no file where none was read', async () => {
    answer('renameCardModelField', { decks: [], cards: 0, notWritten: [] })

    await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, null)

    const dispatched = vi.mocked(cardsService.renameCardModelField).mock.calls[0]?.[0]
    expect(dispatched).toEqual({ path: 'Word.md', from: 'Front', to: 'Template' })
    expect(dispatched).not.toHaveProperty('seen')
  })

  it('answers with the empty string where the service stamped nothing', async () => {
    answer('renameCardModelField', { decks: [], cards: 0, notWritten: [] })

    const renameResult = await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, null)

    expect(renameResult.ok && renameResult.value.at).toBe('')
  })

  it('fails in the service words', async () => {
    answer('renameCardModelField', { error: ErrorCode.MISSING })

    expect(await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, null)).toEqual(
      asFailure('missing'),
    )
  })

  it('is changed where the file moved past what was read', async () => {
    answer('renameCardModelField', { error: ErrorCode.STALE })

    expect(await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, carried)).toEqual(
      asFailure('changed'),
    )
  })
})

describe('making a card model', () => {
  it('asks for the title in the folder with its fields', async () => {
    answer('createCardModel', { path: 'Word.md' })

    await cards.createCardModel('Word', 'Card models', ['Front', 'Back'])

    expect(cardsService.createCardModel).toHaveBeenCalledWith({
      title: 'Word',
      path: 'Card models',
      fields: ['Front', 'Back'],
    })
  })
})

describe('reading a card model', () => {
  it('asks for the path', async () => {
    answer('readCardModel', { cardModel: cardModelMessage, at: stampedAt })

    await cards.readCardModel('Word.md')

    expect(cardsService.readCardModel).toHaveBeenCalledWith({ path: 'Word.md' })
  })

  it('carries the empty string where the service stamped nothing', async () => {
    answer('readCardModel', { cardModel: cardModelMessage })

    const readItem = await cards.readCardModel('Word.md')

    expect(readItem.ok && readItem.value.at).toBe('')
  })

  it('fails in the service words even where a card model came with them', async () => {
    answer('readCardModel', { error: ErrorCode.MISSING, cardModel: cardModelMessage })

    expect(await cards.readCardModel('Word.md')).toEqual(asFailure('missing'))
  })

  it('fails in the service words where no card model came', async () => {
    answer('readCardModel', { error: ErrorCode.UNREADABLE })

    expect(await cards.readCardModel('Word.md')).toEqual(asFailure('unreadable'))
  })

  it('is changed where the file moved past what was requested, though a card model came', async () => {
    answer('readCardModel', { error: ErrorCode.STALE, cardModel: cardModelMessage })

    expect(await cards.readCardModel('Word.md')).toEqual(asFailure('changed'))
  })

  it('is no card model where no card model and no error came', async () => {
    answer('readCardModel', {})

    expect(await cards.readCardModel('Word.md')).toEqual(asFailure('notACardModel'))
  })
})

describe('writing a card model', () => {
  const cardModel = {
    fields: ['Front', 'Back'] as const,
    preamble: 'about',
    tail: 'after',
    templates: [{ name: 'Reading', preamble: 'p', front: '{{Front}}', back: '{{Back}}' }],
  }

  it('sends the fields, the templates and the file it was read from', async () => {
    answer('writeCardModel', { at: stampedAt })

    await cards.writeCardModel('Word.md', cardModel, carried)

    expect(cardsService.writeCardModel).toHaveBeenCalledWith({
      path: 'Word.md',
      fields: ['Front', 'Back'],
      preamble: 'about',
      templates: [{ name: 'Reading', preamble: 'p', front: '{{Front}}', back: '{{Back}}' }],
      tail: 'after',
      baseline: seenAt,
    })
  })

  it('sends no file where none was read', async () => {
    answer('writeCardModel', {})

    await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, null)

    expect(vi.mocked(cardsService.writeCardModel).mock.calls[0]?.[0]).not.toHaveProperty('seen')
  })

  it('answers with the empty string where the service stamped nothing', async () => {
    answer('writeCardModel', {})

    await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, null)

    expect(
      await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, null),
    ).toEqual(asValue({ at: '' }))
  })

  it('fails in the service words', async () => {
    answer('writeCardModel', { error: ErrorCode.ALREADY_EXISTS })

    expect(
      await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, null),
    ).toEqual(asFailure('alreadyExists'))
  })

  it('is changed where the file moved past what was read', async () => {
    answer('writeCardModel', { error: ErrorCode.STALE })

    expect(
      await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, carried),
    ).toEqual(asFailure('changed'))
  })
})
