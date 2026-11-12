/**
 * What the window sends the cards service for each thing it does with a deck or
 * a card model, and what it makes of the answer: an error keeps its own words, a
 * file moved on is `changed`, and a missing stamp is the empty string.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/plugin-sdk'
import { cards } from './cards'
import { ErrorCode } from '../lib/wire/notes'

const wire = vi.hoisted(() => ({ call: vi.fn(), replies: new Map<string, unknown>() }))

vi.mock('@numen/plugin-sdk', async (original) => ({
  ...(await original<typeof import('@numen/plugin-sdk')>()),
  createApi: () => ({ call: wire.call, stream: vi.fn() }),
}))

/** A file the service stamped, and the one string the window carries it as. */
const stampedAt = { path: 'Deck.md', size: 12, mtime: '34' }
const carried = '12 34 Deck.md'

/** A file the window read once, as the service takes it back. */
const seenAt = { path: 'Deck.md', size: 12, mtime: '34' }

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

const answer = (method: string, reply: unknown) => wire.replies.set(method, reply)

const requestOf = (method: string) => wire.call.mock.calls.find((sent) => sent[0] === method)?.[1]

beforeEach(() => {
  wire.replies.clear()
  wire.call.mockReset()
  wire.call.mockImplementation(async (method: string) => asValue(wire.replies.get(method)))
})

describe('making a deck', () => {
  it('asks for the title in the folder', async () => {
    answer('CreateDeck', { path: 'Decks/Words.md' })

    await cards.createDeck('Words', 'Decks')

    expect(requestOf('CreateDeck')).toEqual({ title: 'Words', path: 'Decks' })
  })
})

describe('reading a deck', () => {
  it('asks for the path', async () => {
    answer('ReadDeck', { deck: deckMessage, at: stampedAt })

    await cards.readDeck('Deck.md')

    expect(requestOf('ReadDeck')).toEqual({ path: 'Deck.md' })
  })

  it('carries its sections as name and preamble only, and the stamp as one string', async () => {
    answer('ReadDeck', { deck: deckMessage, at: stampedAt })

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
    answer('ReadDeck', { deck: deckMessage })

    const readItem = await cards.readDeck('Deck.md')

    expect(readItem.ok && readItem.value.at).toBe('')
  })

  it('fails in the service words even where a deck came with them', async () => {
    answer('ReadDeck', { error: ErrorCode.NOT_FOUND, deck: deckMessage, bound: 400 })

    expect(await cards.readDeck('Deck.md')).toEqual(asFailure({ code: 'notFound', bound: 400 }))
  })

  it('fails in the service words where no deck came', async () => {
    answer('ReadDeck', { error: ErrorCode.TOO_LARGE, bound: 900 })

    expect(await cards.readDeck('Deck.md')).toEqual(asFailure({ code: 'tooLarge', bound: 900 }))
  })

  it('is no deck where no deck and no error came', async () => {
    answer('ReadDeck', { bound: 400 })

    expect(await cards.readDeck('Deck.md')).toEqual(asFailure({ code: 'notADeck', bound: 400 }))
  })

  it('is changed where the file moved past what was requested', async () => {
    answer('ReadDeck', { error: ErrorCode.CONFLICT, bound: 400 })

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
    answer('WriteDeck', { at: stampedAt })

    await cards.writeDeck('Deck.md', deck, carried)

    expect(requestOf('WriteDeck')).toEqual({
      path: 'Deck.md',
      preamble: 'about',
      cards: [],
      sections: [{ name: 'Physics', preamble: 'on motion' }],
      tail: 'after',
      baseline: seenAt,
    })
  })

  it('answers with the stamp of what it wrote', async () => {
    answer('WriteDeck', { at: stampedAt })

    expect(await cards.writeDeck('Deck.md', deck, null)).toEqual(asValue({ at: carried }))
  })

  it('answers with the empty string where the service stamped nothing', async () => {
    answer('WriteDeck', {})

    expect(await cards.writeDeck('Deck.md', deck, null)).toEqual(asValue({ at: '' }))
  })

  it('fails in the service words, with the bound', async () => {
    answer('WriteDeck', { error: ErrorCode.DECK_TOO_LARGE, bound: 700 })

    expect(await cards.writeDeck('Deck.md', deck, null)).toEqual(
      asFailure({ code: 'deckTooLarge', bound: 700 }),
    )
  })
})

describe('the card models of a vault', () => {
  it('are requested for up to the limit given', async () => {
    answer('ListCardModels', { cardModels: [], total: 0 })

    await cards.cardModels(5)

    expect(requestOf('ListCardModels')).toEqual({ limit: 5 })
  })

  it('are requested for up to no limit where none is given', async () => {
    answer('ListCardModels', { cardModels: [], total: 0 })

    await cards.cardModels()

    expect(requestOf('ListCardModels')).toEqual({ limit: 0 })
  })
})

describe('renaming a field', () => {
  it('sends the path, both names and the file it was read from', async () => {
    answer('RenameCardModelField', { decks: [], cards: 0, notWritten: [], at: stampedAt })

    await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, carried)

    expect(requestOf('RenameCardModelField')).toEqual({
      path: 'Word.md',
      from: 'Front',
      to: 'Template',
      baseline: seenAt,
    })
  })

  it('sends no file where none was read', async () => {
    answer('RenameCardModelField', { decks: [], cards: 0, notWritten: [] })

    await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, null)

    const dispatched = requestOf('RenameCardModelField')
    expect(dispatched).toEqual({ path: 'Word.md', from: 'Front', to: 'Template' })
    expect(dispatched).not.toHaveProperty('seen')
  })

  it('answers with the empty string where the service stamped nothing', async () => {
    answer('RenameCardModelField', { decks: [], cards: 0, notWritten: [] })

    const renameResult = await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, null)

    expect(renameResult.ok && renameResult.value.at).toBe('')
  })

  it('fails in the service words', async () => {
    answer('RenameCardModelField', { error: ErrorCode.NOT_FOUND })

    expect(await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, null)).toEqual(
      asFailure('notFound'),
    )
  })

  it('is changed where the file moved past what was read', async () => {
    answer('RenameCardModelField', { error: ErrorCode.CONFLICT })

    expect(await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, carried)).toEqual(
      asFailure('changed'),
    )
  })
})

describe('making a card model', () => {
  it('asks for the title in the folder with its fields', async () => {
    answer('CreateCardModel', { path: 'Word.md' })

    await cards.createCardModel('Word', 'Card models', ['Front', 'Back'])

    expect(requestOf('CreateCardModel')).toEqual({
      title: 'Word',
      path: 'Card models',
      fields: ['Front', 'Back'],
    })
  })
})

describe('reading a card model', () => {
  it('asks for the path', async () => {
    answer('ReadCardModel', { cardModel: cardModelMessage, at: stampedAt })

    await cards.readCardModel('Word.md')

    expect(requestOf('ReadCardModel')).toEqual({ path: 'Word.md' })
  })

  it('carries the empty string where the service stamped nothing', async () => {
    answer('ReadCardModel', { cardModel: cardModelMessage })

    const readItem = await cards.readCardModel('Word.md')

    expect(readItem.ok && readItem.value.at).toBe('')
  })

  it('fails in the service words even where a card model came with them', async () => {
    answer('ReadCardModel', { error: ErrorCode.NOT_FOUND, cardModel: cardModelMessage })

    expect(await cards.readCardModel('Word.md')).toEqual(asFailure('notFound'))
  })

  it('fails in the service words where no card model came', async () => {
    answer('ReadCardModel', { error: ErrorCode.UNREADABLE })

    expect(await cards.readCardModel('Word.md')).toEqual(asFailure('unreadable'))
  })

  it('is changed where the file moved past what was requested, though a card model came', async () => {
    answer('ReadCardModel', { error: ErrorCode.CONFLICT, cardModel: cardModelMessage })

    expect(await cards.readCardModel('Word.md')).toEqual(asFailure('changed'))
  })

  it('is no card model where no card model and no error came', async () => {
    answer('ReadCardModel', {})

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
    answer('WriteCardModel', { at: stampedAt })

    await cards.writeCardModel('Word.md', cardModel, carried)

    expect(requestOf('WriteCardModel')).toEqual({
      path: 'Word.md',
      fields: ['Front', 'Back'],
      preamble: 'about',
      templates: [{ name: 'Reading', preamble: 'p', front: '{{Front}}', back: '{{Back}}' }],
      tail: 'after',
      baseline: seenAt,
    })
  })

  it('sends no file where none was read', async () => {
    answer('WriteCardModel', {})

    await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, null)

    expect(requestOf('WriteCardModel')).not.toHaveProperty('seen')
  })

  it('answers with the empty string where the service stamped nothing', async () => {
    answer('WriteCardModel', {})

    await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, null)

    expect(
      await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, null),
    ).toEqual(asValue({ at: '' }))
  })

  it('fails in the service words', async () => {
    answer('WriteCardModel', { error: ErrorCode.ALREADY_EXISTS })

    expect(
      await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, null),
    ).toEqual(asFailure('alreadyExists'))
  })

  it('is changed where the file moved past what was read', async () => {
    answer('WriteCardModel', { error: ErrorCode.CONFLICT })

    expect(
      await cards.writeCardModel('Word.md', { ...cardModel, fields: ['Front'] }, carried),
    ).toEqual(asFailure('changed'))
  })
})
