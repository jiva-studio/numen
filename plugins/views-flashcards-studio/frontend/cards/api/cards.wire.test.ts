/**
 * The vault's decks and card models, read by the client the schema generates.
 *
 * What is proved here is the crossing: a field the answer leaves at its default
 * arrives as the default, a section a card is placed under arrives as a number or
 * as nothing at all, and a file the caller read goes back out as the three
 * parts the schema holds it in.
 */
import { describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/plugin-sdk'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

/** What the window asked for, as the transport wrote it out. */
let request: Record<string, unknown>[] = []

/** What the application answers with, in the words the schema writes it in. */
const replyWith = (answer: unknown) => {
  request = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (_url: string, init: { body: string }) => {
      request.push(JSON.parse(init.body))
      return new Response(JSON.stringify(answer), {
        headers: { 'content-type': 'application/json' },
      })
    }),
  )
}

/** A file the window read, in the one string it carries it as. */
const readItem = '12 34 Deck.md'

const { cards } = await import('./cards')

describe('the card models of a vault', () => {
  it('come back with how many the vault holds', async () => {
    replyWith({
      cardModels: [{ path: 'Word.md', title: 'Word', fields: ['Front', 'Back'] }],
      total: 3,
    })

    expect(await cards.cardModels(1)).toEqual({
      cardModels: [{ path: 'Word.md', title: 'Word', fields: ['Front', 'Back'] }],
      held: 3,
    })
  })

  it('are requested for without a ceiling where none was named', async () => {
    replyWith({ cardModels: [], total: 0 })

    await cards.cardModels()

    expect(request[0]).toEqual({ limit: 0 })
  })
})

describe('making a deck', () => {
  it('answers where it was filed', async () => {
    replyWith({ path: 'Decks/Words.md' })

    expect(await cards.createDeck('Words', 'Decks')).toEqual({
      ok: true,
      value: { path: 'Decks/Words.md' },
    })
  })

  it('carries the error in the words the window uses', async () => {
    replyWith({ ok: false, error: 'already_exists' })

    const createdItem = await cards.createDeck('Words', 'Decks')
    expect(createdItem.ok ? null : createdItem.error).toBe('alreadyExists')
  })
})

describe('making a card model', () => {
  it('carries the fields it is cut with', async () => {
    replyWith({ path: 'Word.md' })

    expect(await cards.createCardModel('Word', '', ['Front', 'Back'])).toEqual({
      ok: true,
      value: { path: 'Word.md' },
    })
    expect(request[0]).toEqual({ title: 'Word', path: '', fields: ['Front', 'Back'] })
  })
})

describe('reading a deck', () => {
  it('carries every card, and the section as a number or as nothing', async () => {
    replyWith({
      deck: {
        path: 'Deck.md',
        title: 'Deck',
        preamble: 'about this deck',
        cards: [
          {
            cardId: 'a1',
            sectionIndex: 0,
            heading: 'Entropy',
            cardModelLink: 'Word',
            cardModelPath: 'Word.md',
            preamble: 'a note',
            values: [{ field: 'Front', text: '<p>Entropy</p>' }],
          },
          {
            cardId: 'a2',
            heading: 'Order',
            cardModelLink: 'Word',
            cardModelPath: 'Word.md',
            preamble: '',
            values: [],
          },
        ],
        sections: [{ name: 'Physics', preamble: '' }],
        tail: 'after',
        problems: [],
      },
      at: { path: 'Deck.md', size: 12, mtime: '34' },
      bound: 400,
    })

    const answer = await cards.readDeck('Deck.md')

    if (!answer.ok) throw new Error('the deck failed')
    expect(answer.value.deck.cards.map((one) => one.sectionIndex)).toEqual([0, null])
    expect(answer.value.deck.cards[0]?.values).toEqual([{ field: 'Front', text: '<p>Entropy</p>' }])
    expect(answer.value.at).toBe(readItem)
  })

  it('is no deck where the answer carries none', async () => {
    replyWith({ error: 'not_a_deck' })

    const answer = await cards.readDeck('Notes.md')

    expect(answer.ok).toBe(false)
    expect(answer.ok ? null : answer.error.code).toBe('notADeck')
  })

  it('names what is wrong with it in the words the window uses', async () => {
    replyWith({
      deck: {
        path: 'Deck.md',
        title: 'Deck',
        preamble: '',
        cards: [],
        sections: [],
        tail: '',
        problems: [
          { kind: 'card_id_carried_twice', card: 2, field: '', text: 'a1' },
          { kind: 'card_without_a_card_model', field: 'Front', text: '' },
        ],
      },
    })

    const readItem = await cards.readDeck('Deck.md')
    expect(readItem.ok ? readItem.value.deck.problems : null).toEqual([
      { kind: 'cardIdCarriedTwice', card: 2, template: null, field: '', text: 'a1' },
      { kind: 'cardWithoutACardModel', card: null, template: null, field: 'Front', text: '' },
    ])
  })
})

describe('writing a deck', () => {
  it('takes the file the window read back apart into what the schema holds', async () => {
    replyWith({ at: { path: 'Deck.md', size: 12, mtime: '34' }, bound: 400 })

    await cards.writeDeck(
      'Deck.md',
      {
        preamble: 'about',
        tail: 'after',
        cards: [
          {
            cardId: 'a1',
            sectionIndex: null,
            heading: 'Entropy',
            cardModelLink: 'Word',
            cardModelPath: 'Word.md',
            preamble: '',
            values: [],
          },
        ],
        sections: [{ name: 'Physics', preamble: '' }],
      },
      readItem,
    )

    expect(request[0]?.baseline).toEqual({ path: 'Deck.md', size: 12, mtime: '34' })
    expect(request[0]?.cards).toEqual([
      { cardId: 'a1', heading: 'Entropy', cardModelLink: 'Word', preamble: '', values: [] },
    ])
  })

  it('names no file where the window read none', async () => {
    replyWith({})

    await cards.writeDeck('Deck.md', { preamble: '', tail: '', cards: [], sections: [] }, null)

    expect(request[0]?.baseline).toBeUndefined()
  })

  it('says the file moved past what the window read', async () => {
    replyWith({ error: 'stale' })

    const answer = await cards.writeDeck(
      'Deck.md',
      { preamble: '', tail: '', cards: [], sections: [] },
      readItem,
    )

    expect(answer.ok ? null : answer.error.code).toBe('changed')
  })
})

describe('a card model', () => {
  it('arrives back with every template it shows a card through', async () => {
    replyWith({
      cardModel: {
        path: 'Word.md',
        title: 'Word',
        fields: ['Front', 'Back'],
        preamble: '',
        templates: [{ name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' }],
        tail: '',
        problems: [],
      },
      at: { path: 'Word.md', size: 12, mtime: '34' },
    })

    const answer = await cards.readCardModel('Word.md')
    expect(answer.ok ? answer.value.cardModel.templates : null).toEqual([
      { name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' },
    ])
  })

  it('is no card model where the answer carries none', async () => {
    replyWith({ error: 'not_a_card_model' })

    expect(await cards.readCardModel('Notes.md')).toEqual(asFailure('notACardModel'))
  })

  it('is written with its fields and its templates', async () => {
    replyWith({ at: { path: 'Word.md', size: 12, mtime: '34' } })

    const answer = await cards.writeCardModel(
      'Word.md',
      {
        fields: ['Front', 'Back'],
        preamble: 'about',
        tail: '',
        templates: [{ name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' }],
      },
      readItem,
    )

    expect(request[0]?.fields).toEqual(['Front', 'Back'])
    expect(answer.ok && answer.value.at).toBe('12 34 Word.md')
  })
})

describe('renaming a field', () => {
  it('counts what it reached and names what it did not', async () => {
    replyWith({
      decks: ['Words.md', 'Roots.md'],
      cards: 12,
      notWritten: [{ path: 'Old.md', problem: { kind: 'field_not_renamed', text: 'Front' } }],
      at: { path: 'Word.md', size: 12, mtime: '34' },
    })

    expect(await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, readItem)).toEqual(
      asValue({
        decks: ['Words.md', 'Roots.md'],
        cards: 12,
        notWritten: [{ path: 'Old.md', text: 'Front' }],
        at: '12 34 Word.md',
      }),
    )
  })

  it('leaves a deck it could not read carrying nothing to say', async () => {
    replyWith({ decks: [], cards: 0, notWritten: [{ path: 'Old.md' }] })

    const answer = await cards.renameField('Word.md', { from: 'Front', to: 'Template' }, null)
    expect(answer.ok ? answer.value.notWritten : null).toEqual([{ path: 'Old.md', text: '' }])
  })
})
