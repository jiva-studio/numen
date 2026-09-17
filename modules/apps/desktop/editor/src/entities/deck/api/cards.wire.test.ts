/**
 * The vault's decks and stencils, read by the client the schema generates.
 *
 * What is proved here is the crossing: a field the answer leaves at its default
 * arrives as the default, a section a card stands under arrives as a number or
 * as nothing at all, and a file the caller read goes back out as the three
 * parts the schema holds it in.
 */
import { describe, expect, it, vi } from 'vitest'
import { asFailure } from '@numen/wire'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

/** What the window asked for, as the transport wrote it out. */
let asked: Record<string, unknown>[] = []

/** What the application answers with, in the words the schema writes it in. */
const replyWith = (answer: unknown) => {
  asked = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (_url: string, init: { body: Uint8Array }) => {
      asked.push(JSON.parse(new TextDecoder().decode(init.body)))
      return new Response(JSON.stringify(answer), {
        headers: { 'content-type': 'application/json' },
      })
    }),
  )
}

/** A file the window read, in the one string it carries it as. */
const read = '12 34 Deck.md'

const { cards } = await import('./cards')

describe('the stencils of a vault', () => {
  it('come back with how many the vault holds', async () => {
    replyWith({
      stencils: [{ path: 'Word.md', title: 'Word', fields: ['Front', 'Back'] }],
      total: 3,
    })

    expect(await cards.stencils(1)).toEqual({
      stencils: [{ path: 'Word.md', title: 'Word', fields: ['Front', 'Back'] }],
      held: 3,
    })
  })

  it('are asked for without a ceiling where none was named', async () => {
    replyWith({ stencils: [], total: 0 })

    await cards.stencils()

    expect(asked[0]).toEqual({})
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
    replyWith({ ok: false, error: 'ERROR_CODE_OCCUPIED' })

    const made = await cards.createDeck('Words', 'Decks')
    expect(made.ok ? null : made.error).toBe('occupied')
  })
})

describe('making a stencil', () => {
  it('carries the fields it is cut with', async () => {
    replyWith({ path: 'Word.md' })

    expect(await cards.createStencil('Word', '', ['Front', 'Back'])).toEqual({
      ok: true,
      value: { path: 'Word.md' },
    })
    expect(asked[0]).toEqual({ title: 'Word', fields: ['Front', 'Back'] })
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
            mark: 'a1',
            sectionIndex: 0,
            heading: 'Entropy',
            stencilLink: 'Word',
            stencilPath: 'Word.md',
            preamble: 'a note',
            values: [{ field: 'Front', text: '<p>Entropy</p>' }],
          },
          { mark: 'a2', heading: 'Order', stencilLink: 'Word', stencilPath: 'Word.md' },
        ],
        sections: [{ name: 'Physics', preamble: '' }],
        tail: 'after',
        problems: [],
      },
      at: { path: 'Deck.md', size: '12', mtime: '34' },
      bound: '400',
    })

    const answer = await cards.readDeck('Deck.md')

    if (!answer.ok) throw new Error('the deck was refused')
    expect(answer.value.deck.cards.map((one) => one.sectionIndex)).toEqual([0, null])
    expect(answer.value.deck.cards[0]?.values).toEqual([{ field: 'Front', text: '<p>Entropy</p>' }])
    expect(answer.value.at).toBe(read)
  })

  it('is no deck where the answer carries none', async () => {
    replyWith({ error: 'ERROR_CODE_NOT_A_DECK' })

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
          { fault: 'FAULT_MARK_CARRIED_TWICE', card: 2, field: '', text: 'a1' },
          { fault: 'FAULT_CARD_WITHOUT_A_STENCIL', field: 'Front', text: '' },
        ],
      },
    })

    const read = await cards.readDeck('Deck.md')
    expect(read.ok ? read.value.deck.problems : null).toEqual([
      { fault: 'markCarriedTwice', card: 2, face: null, field: '', text: 'a1' },
      { fault: 'cardWithoutAStencil', card: null, face: null, field: 'Front', text: '' },
    ])
  })
})

describe('writing a deck', () => {
  it('takes the file the window read back apart into what the schema holds', async () => {
    replyWith({ at: { path: 'Deck.md', size: '12', mtime: '34' }, bound: '400' })

    await cards.writeDeck(
      'Deck.md',
      {
        preamble: 'about',
        tail: 'after',
        cards: [
          {
            mark: 'a1',
            sectionIndex: null,
            heading: 'Entropy',
            stencilLink: 'Word',
            stencilPath: 'Word.md',
            preamble: '',
            values: [],
          },
        ],
        sections: [{ name: 'Physics', preamble: '' }],
      },
      read,
    )

    expect(asked[0]?.seen).toEqual({ path: 'Deck.md', size: '12', mtime: '34' })
    expect(asked[0]?.cards).toEqual([{ mark: 'a1', heading: 'Entropy', stencilLink: 'Word' }])
  })

  it('names no file where the window read none', async () => {
    replyWith({})

    await cards.writeDeck('Deck.md', { preamble: '', tail: '', cards: [], sections: [] }, null)

    expect(asked[0]?.seen).toBeUndefined()
  })

  it('says the file moved past what the window read', async () => {
    replyWith({ error: 'ERROR_CODE_STALE' })

    const answer = await cards.writeDeck(
      'Deck.md',
      { preamble: '', tail: '', cards: [], sections: [] },
      read,
    )

    expect(answer.ok ? null : answer.error.code).toBe('changed')
  })
})

describe('a stencil', () => {
  it('comes back with every face it shows a card through', async () => {
    replyWith({
      stencil: {
        path: 'Word.md',
        title: 'Word',
        fields: ['Front', 'Back'],
        preamble: '',
        faces: [{ name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' }],
        tail: '',
        problems: [],
      },
      at: { path: 'Word.md', size: '12', mtime: '34' },
    })

    const answer = await cards.readStencil('Word.md')
    expect(answer.ok ? answer.value.stencil.faces : null).toEqual([
      { name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' },
    ])
  })

  it('is no stencil where the answer carries none', async () => {
    replyWith({ error: 'ERROR_CODE_NOT_A_STENCIL' })

    expect(await cards.readStencil('Notes.md')).toEqual(asFailure('notAStencil'))
  })

  it('is written with its fields and its faces', async () => {
    replyWith({ at: { path: 'Word.md', size: '12', mtime: '34' } })

    const answer = await cards.writeStencil(
      'Word.md',
      ['Front', 'Back'],
      {
        preamble: 'about',
        tail: '',
        faces: [{ name: 'Reading', preamble: '', front: '{{Front}}', back: '{{Back}}' }],
      },
      read,
    )

    expect(asked[0]?.fields).toEqual(['Front', 'Back'])
    expect(answer.ok && answer.value.at).toBe('12 34 Word.md')
  })
})

describe('renaming a field', () => {
  it('counts what it reached and names what it did not', async () => {
    replyWith({
      decks: ['Words.md', 'Roots.md'],
      cards: 12,
      notWritten: [
        { path: 'Old.md', problem: { fault: 'FAULT_FIELD_NOT_RENAMED', text: 'Front' } },
      ],
      at: { path: 'Word.md', size: '12', mtime: '34' },
    })

    expect(await cards.renameField('Word.md', 'Front', 'Face', read)).toEqual({
      decks: ['Words.md', 'Roots.md'],
      cards: 12,
      notWritten: [{ path: 'Old.md', text: 'Front' }],
      error: null,
      changed: false,
      at: '12 34 Word.md',
    })
  })

  it('leaves a deck it could not read carrying nothing to say', async () => {
    replyWith({ decks: [], cards: 0, notWritten: [{ path: 'Old.md' }] })

    expect((await cards.renameField('Word.md', 'Front', 'Face', null)).notWritten).toEqual([
      { path: 'Old.md', text: '' },
    ])
  })
})
