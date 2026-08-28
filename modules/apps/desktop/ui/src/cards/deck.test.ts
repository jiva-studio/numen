/**
 * What a deck tab holds, asked without a screen: what it reads, what it writes
 * back, and what it does when the file moved past what it read.
 */
import { describe, expect, it } from 'vitest'
import type { Cards, Carded, Problem, Refused } from '../core'
import { putting } from '../putting'
import { windowing } from '../windowing'
import { DECK } from '../workspace'
import { decking, type Held } from './deck'
import { WORDS as words } from './words'

/** The one place a file is opened from. Nothing here opens one. */
const puts = () => putting({ types: async () => new Map() })

/** A moment for whatever the tab asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

// The two cards name one stencil two ways: by a path from the root, and by a
// name carrying an alias. Both reach the file the vault says they reach.
const CARDS: readonly Carded[] = [
  {
    name: 'Llama',
    stencil: 'cards/Animal',
    stencilAt: 'Animal.md',
    lead: '',
    values: [{ field: 'Height', text: 'about 45"' }],
  },
  { name: 'Alpaca', stencil: 'Animal|зверь', stencilAt: 'Animal.md', lead: '', values: [] },
]

/** A vault holding one deck, writing down every write it was asked for. */
const vault = (
  answers: {
    refusal?: Refused
    problems?: readonly Problem[]
    bound?: number
    /** The write answers that the file moved past what the tab read. */
    changed?: boolean
    /** The cards the file holds, where a test wants other ones. */
    cards?: readonly Carded[]
  } = {},
) => {
  const written: string[] = []
  const seen: (string | null)[] = []
  let reads = 0
  let cards: readonly Carded[] = answers.cards ?? CARDS

  let listed = 0

  const core: Cards = {
    stencils: async () => {
      listed += 1
      return {
        stencils: [{ path: 'Animal.md', title: 'Animal', fields: ['Name', 'Height'] }],
        held: 1,
      }
    },
    makeDeck: async (title) => ({ path: `${title}.md`, refusal: null }),
    makeStencil: async (title) => ({ path: `${title}.md`, refusal: null }),
    renameField: async () => ({
      decks: [],
      cards: 0,
      notWritten: [],
      refusal: null,
      changed: false,
      at: '',
    }),
    readDeck: async (path) => {
      reads += 1
      if (answers.refusal) {
        return { deck: null, refusal: answers.refusal, at: '', bound: answers.bound ?? 0 }
      }
      return {
        deck: {
          path,
          title: 'Animals',
          preamble: '',
          cards,
          tail: '',
          problems: answers.problems ?? [],
        },
        refusal: null,
        at: `read ${reads}`,
        bound: 0,
      }
    },
    writeDeck: async (path, deck, presented) => {
      written.push(`${path} ${deck.cards.map((card) => card.name).join(', ') || '—'}`)
      seen.push(presented)
      if (answers.changed) return { refusal: null, changed: true, at: '', bound: 0 }
      cards = deck.cards
      return { refusal: null, changed: false, at: 'written', bound: 0 }
    },
    readStencil: async () => ({ stencil: null, refusal: 'missing', at: '' }),
    writeStencil: async () => ({ refusal: null, changed: false, at: '' }),
  }

  return {
    core,
    written,
    seen,
    reads: () => reads,
    listed: () => listed,
    /** The file written from somewhere else, which the next read answers with. */
    holds: (next: readonly Carded[]) => {
      cards = next
    },
  }
}

/** A window with one deck open on a file, and what that tab holds. */
const open = async (
  answers: Parameters<typeof vault>[0] = {},
  path = 'Animals.md',
) => {
  const one = vault(answers)
  const held = windowing()
  const decks = decking(one.core, held.host, puts())
  held.declares([decks.kind])
  const id = await held.opens(DECK, path)
  await settles()
  const tab = held.host.holds<Held>(DECK, id) as Held
  return { ...one, held, decks, id, tab }
}

describe('a deck opened', () => {
  it('draws the cards the vault read, in the order the file had them', async () => {
    const { tab } = await open()

    expect(tab.deck().cards.map((card) => card.name)).toStrictEqual(['Llama', 'Alpaca'])
  })

  it('is called what the file is called', async () => {
    const { decks, id, held } = await open()

    expect(decks.kind.called(held.host.holds<Held>(DECK, id) as Held)).toBe('Animals')
  })

  it('offers every stencil the vault holds as a cut', async () => {
    const { tab } = await open()

    expect(tab.cuts()).toStrictEqual([{ name: 'Animal', fields: ['Name', 'Height'] }])
  })

  it('draws each card under the stencil its wikilink reached', async () => {
    const { tab } = await open()

    expect(tab.drawn().map((card) => card.stencil)).toStrictEqual(['Animal', 'Animal'])
  })

  it('is one tab per file, so the same deck asked for twice is the tab it has', async () => {
    const { held, id } = await open()

    const again = await held.opens(DECK, 'Animals.md')

    expect(again).toBe(id)
  })

  it('carries no mark while nothing is unwritten', async () => {
    const { decks, tab } = await open()

    expect(decks.kind.marked?.(tab)).toBeUndefined()
  })
})

describe('a card written in a deck', () => {
  it('stands in the deck the tab draws', async () => {
    const { tab } = await open()

    tab.adds('Vicuña', 'Animal', [{ field: 'Height', text: '' }])

    expect(tab.deck().cards.map((card) => card.name)).toStrictEqual([
      'Llama',
      'Alpaca',
      'Vicuña',
    ])
  })

  it('stands under where the vault files the stencil it was cut by', async () => {
    const { tab } = await open()

    tab.adds('Vicuña', 'Animal', [])

    expect(tab.deck().cards.at(-1)?.stencilAt).toBe('Animal.md')
  })

  it('leaves the tab unsaved, and marks it', async () => {
    const { decks, tab } = await open()

    tab.adds('Vicuña', 'Animal', [])

    expect(tab.shown().state).toBe('unsaved')
    expect(decks.kind.marked?.(tab)).toBe('unsaved')
  })

  it('reaches the vault as the cards of that file, with the file it was read from', async () => {
    const { decks, tab, written, seen } = await open()

    tab.adds('Vicuña', 'Animal', [])
    await decks.flush()

    expect(written).toStrictEqual(['Animals.md Llama, Alpaca, Vicuña'])
    expect(seen).toStrictEqual(['read 1'])
  })

  it('writes nothing where nothing was touched', async () => {
    const { decks, written } = await open()

    await decks.flush()

    expect(written).toStrictEqual([])
  })
})

describe('a deck whose file moved past what was read', () => {
  it('is overtaken once the write comes back saying the file changed', async () => {
    const { decks, tab } = await open({ changed: true })

    tab.adds('Vicuña', 'Animal', [])
    await decks.flush()

    expect(tab.shown().state).toBe('overtaken')
  })

  it('keeps what the person wrote when they say so, over whatever the file holds', async () => {
    const { decks, tab, written, seen } = await open({ changed: true })

    tab.adds('Vicuña', 'Animal', [])
    await decks.flush()
    tab.keep()
    await settles()

    expect(written).toHaveLength(2)
    expect(seen[1]).toBeNull()
  })

  it('reads the file again when the person takes what it holds', async () => {
    const { decks, tab, reads } = await open({ changed: true })

    tab.adds('Vicuña', 'Animal', [])
    await decks.flush()
    tab.take()
    await settles()

    expect(reads()).toBe(2)
    expect(tab.shown().state).toBe('clean')
    expect(tab.deck().cards.map((card) => card.name)).toStrictEqual(['Llama', 'Alpaca'])
  })
})

describe('a deck the vault refused', () => {
  it('says which bound it is over, and draws no card', async () => {
    const { tab } = await open({ refusal: 'deckTooLarge', bound: 8388608 })

    expect(tab.saying()).toBe(words.tooLarge(8388608))
    expect(tab.deck().cards).toStrictEqual([])
  })

  it('says the note is not a deck where that is what it is', async () => {
    const { tab } = await open({ refusal: 'notADeck' })

    expect(tab.saying()).toBe(words.notADeck)
  })

  it('says nothing where the deck was read', async () => {
    const { tab } = await open()

    expect(tab.saying()).toBe('')
  })
})

describe('what is wrong with a deck', () => {
  const nameless: Problem = {
    fault: 'cardWithoutAName',
    card: 1,
    face: null,
    field: '',
    text: 'a card with no name',
  }

  it('stands against the card it was read against', async () => {
    const { tab } = await open({ problems: [nameless] })

    const marks = tab.marks()
    const second = tab.deck().cards[1]?.id ?? ''
    expect(marks.at.get(second)).toStrictEqual(['a card with no name'])
  })

  it('stands against no other card', async () => {
    const { tab } = await open({ problems: [nameless] })

    const first = tab.deck().cards[0]?.id ?? ''
    expect(tab.marks().at.has(first)).toBe(false)
  })

  it('is nothing at all where the vault reported none', async () => {
    const { tab } = await open()

    expect(tab.marks().at.size).toBe(0)
    expect(tab.marks().whole).toStrictEqual([])
  })
})

describe('the field a card is named by, written over', () => {
  // One stencil named three ways: by a path, by a name carrying an alias, and
  // by the identifier of the note.
  const WRITTEN = ['cards/Animal', 'Animal|зверь', 'note://01J3ZQ8W0T7K9V2M4N6P8R0S1T']

  /** A deck of one card, under the wikilink it wrote for its stencil. */
  const only = (stencil: string, stencilAt = 'Animal.md'): readonly Carded[] => [
    { name: 'Llama', stencil, stencilAt, lead: '', values: [] },
  ]

  it('renames the card, whatever the card wrote in its brackets', async () => {
    for (const one of WRITTEN) {
      const { tab } = await open({ cards: only(one) })
      const card = tab.deck().cards[0]?.id ?? ''

      tab.writes(card, 'Name', 'Vicuña')

      expect(tab.deck().cards[0]?.name).toBe('Vicuña')
    }
  })

  it('stands under no heading of its own, so no card carries that field twice', async () => {
    for (const one of WRITTEN) {
      const { tab } = await open({ cards: only(one) })
      const card = tab.deck().cards[0]?.id ?? ''

      tab.writes(card, 'Name', 'Vicuña')

      expect(tab.deck().cards[0]?.values).toStrictEqual([])
    }
  })

  it('is a value where the field is another one the stencil declares', async () => {
    const { tab } = await open({ cards: only('cards/Animal') })
    const card = tab.deck().cards[0]?.id ?? ''

    tab.writes(card, 'Height', 'about 45"')

    expect(tab.deck().cards[0]?.name).toBe('Llama')
    expect(tab.deck().cards[0]?.values).toStrictEqual([{ field: 'Height', text: 'about 45"' }])
  })

  it('is a value for a card whose link reached no stencil', async () => {
    const { tab } = await open({ cards: only('Gone', '') })
    const card = tab.deck().cards[0]?.id ?? ''

    tab.writes(card, 'Name', 'Vicuña')

    expect(tab.deck().cards[0]?.name).toBe('Llama')
    expect(tab.deck().cards[0]?.values).toStrictEqual([{ field: 'Name', text: 'Vicuña' }])
  })
})

describe('the vault changing under the window', () => {
  it('asks for no stencil while the window holds no deck', async () => {
    const one = vault()
    const held = windowing()
    const decks = decking(one.core, held.host, puts())
    held.declares([decks.kind])

    decks.changed(['Notes.md'])
    await settles()

    expect(one.listed()).toBe(0)
  })

  it('lists them again once a deck is open', async () => {
    const { decks, listed } = await open()

    decks.changed(['Notes.md'])
    await settles()

    expect(listed()).toBe(2)
  })
})

describe('a deck read again under the window', () => {
  /** Everything the tab hands the grid to draw. */
  const drawing = (tab: Held) => ({
    deck: tab.deck(),
    drawn: tab.drawn(),
    cuts: tab.cuts(),
    marks: tab.marks(),
  })

  it('leaves what the grid is drawing standing, where the file reads the same', async () => {
    const one = await open()
    const was = drawing(one.tab)

    one.decks.changed(['Animals.md'])
    await settles()

    // Each of them the same thing, and not merely a thing that reads the same:
    // a card under the keyboard is redrawn by anything else.
    expect(drawing(one.tab)).toStrictEqual(was)
    expect(one.tab.deck()).toBe(was.deck)
    expect(one.tab.drawn()).toBe(was.drawn)
    expect(one.tab.cuts()).toBe(was.cuts)
    expect(one.tab.marks()).toBe(was.marks)
  })

  it('draws the file again where it was written from somewhere else', async () => {
    const one = await open()
    const was = drawing(one.tab)

    one.holds([
      { name: 'Vicuña', stencil: 'Animal', stencilAt: 'Animal.md', lead: '', values: [] },
    ])
    one.decks.changed(['Animals.md'])
    await settles()

    expect(one.tab.drawn().map((card) => card.name)).toStrictEqual(['Vicuña'])
    expect(one.tab.deck()).not.toBe(was.deck)
  })
})

