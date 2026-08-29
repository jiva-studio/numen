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
const puts = () => putting({ standing: async () => new Map() })

/** A moment for whatever the tab asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

// The two cards name one stencil two ways: by a path from the root, and by a
// name carrying an alias. Both reach the file the vault says they reach.
const CARDS: readonly Carded[] = [
  {
    mark: 'k7m2xq9fzp',
    section: null,
    heading: 'Llama',
    stencil: 'cards/Animal',
    stencilAt: 'Animal.md',
    lead: '',
    values: [
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 45"' },
    ],
  },
  {
    mark: '3n8vr4tqch',
    section: null,
    heading: 'Alpaca',
    stencil: 'Animal|животное',
    stencilAt: 'Animal.md',
    lead: '',
    values: [{ field: 'Name', text: 'Alpaca' }],
  },
]

/** What a card is called on a screen: the first line of its first field. */
const calling = (card: { values: readonly { field: string; text: string }[] }): string =>
  card.values[0]?.text || '—'

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
    /** The sections the file holds, where a test wants some. */
    sections?: readonly { name: string; lead: string }[]
    /** The vault is out of reach, and a read of the deck reaches nothing. */
    unreachable?: boolean
    /** What a write of the deck is refused for. */
    wrote?: Refused
    /** How many listings of the stencils go unanswered before one answers. */
    unlisted?: number
  } = {},
) => {
  const written: string[] = []
  /** Each deck as the vault was handed it, for what a card carries to be asked. */
  const wrote: Parameters<Cards['writeDeck']>[1][] = []
  const seen: (string | null)[] = []
  let reads = 0
  let cards: readonly Carded[] = answers.cards ?? CARDS

  let listed = 0

  const core: Cards = {
    stencils: async () => {
      listed += 1
      if (listed <= (answers.unlisted ?? 0)) throw new Error('out of reach')
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
      if (answers.unreachable) throw new Error('out of reach')
      if (answers.refusal) {
        return { deck: null, refusal: answers.refusal, at: '', bound: answers.bound ?? 0 }
      }
      return {
        deck: {
          path,
          title: 'Animals',
          preamble: '',
          cards,
          sections: answers.sections ?? [],
          tail: '',
          problems: answers.problems ?? [],
        },
        refusal: null,
        at: `read ${reads}`,
        bound: 0,
      }
    },
    writeDeck: async (path, deck, presented) => {
      written.push(`${path} ${deck.cards.map(calling).join(', ') || '—'}`)
      wrote.push(deck)
      seen.push(presented)
      if (answers.wrote) return { refusal: answers.wrote, changed: false, at: '', bound: 0 }
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
    wrote: () => wrote,
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
  const road = puts()
  const decks = decking(one.core, held.host, road)
  held.declares([decks.kind])
  const id = await held.opens(DECK, path)
  await settles()
  const tab = held.host.holds<Held>(DECK, id) as Held
  return { ...one, held, road, decks, id, tab }
}

describe('a deck opened', () => {
  it('draws the cards the vault read, in the order the file had them', async () => {
    const { tab } = await open()

    expect(tab.deck().cards.map(calling)).toStrictEqual(['Llama', 'Alpaca'])
  })

  it('knows each card by the mark the file carries for it', async () => {
    const { tab } = await open()

    expect(tab.deck().cards.map((card) => card.id)).toStrictEqual(['k7m2xq9fzp', '3n8vr4tqch'])
    expect(tab.drawn().map((card) => card.id)).toStrictEqual(['k7m2xq9fzp', '3n8vr4tqch'])
  })

  it('draws the sections the vault read, each under an identity of its own', async () => {
    const { tab } = await open({ sections: [{ name: 'Roots', lead: '' }] })

    expect(tab.bands().map((band) => band.name)).toStrictEqual(['Roots'])
    expect(tab.bands()[0]?.id).toBeTruthy()
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

    tab.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)

    expect(tab.deck().cards.map(calling)).toStrictEqual(['Llama', 'Alpaca', 'Vicuña'])
  })

  it('stands under where the vault files the stencil it was cut by', async () => {
    const { tab } = await open()

    tab.adds('Animal', [], null)

    expect(tab.deck().cards.at(-1)?.stencilAt).toBe('Animal.md')
  })

  it('carries no mark, which is written where the deck is made whole', async () => {
    const { tab } = await open()

    tab.adds('Animal', [], null)

    expect(tab.deck().cards.at(-1)?.mark).toBe('')
  })

  it('leaves the tab unsaved, and marks it', async () => {
    const { decks, tab } = await open()

    tab.adds('Animal', [], null)

    expect(tab.shown().state).toBe('unsaved')
    expect(decks.kind.marked?.(tab)).toBe('unsaved')
  })

  it('reaches the vault as the cards of that file, with the file it was read from', async () => {
    const { decks, tab, written, seen } = await open()

    tab.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.flush()

    expect(written).toStrictEqual(['Animals.md Llama, Alpaca, Vicuña'])
    expect(seen).toStrictEqual(['read 1'])
  })

  it('reaches the vault carrying the mark and the heading the file gave it', async () => {
    const { decks, tab, wrote } = await open()

    tab.writes('k7m2xq9fzp', 'Height', 1, 'about 46"')
    await decks.flush()

    expect(wrote()[0]?.cards[0]?.mark).toBe('k7m2xq9fzp')
    expect(wrote()[0]?.cards[0]?.heading).toBe('Llama')
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

    tab.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.flush()

    expect(tab.shown().state).toBe('overtaken')
  })

  it('keeps what the person wrote when they say so, over whatever the file holds', async () => {
    const { decks, tab, written, seen } = await open({ changed: true })

    tab.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.flush()
    tab.keep()
    await settles()

    expect(written).toHaveLength(2)
    expect(seen[1]).toBeNull()
  })

  it('reads the file again when the person takes what it holds', async () => {
    const { decks, tab, reads } = await open({ changed: true })

    tab.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.flush()
    tab.take()
    await settles()

    expect(reads()).toBe(2)
    expect(tab.shown().state).toBe('clean')
    expect(tab.deck().cards.map(calling)).toStrictEqual(['Llama', 'Alpaca'])
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
  const stencilless: Problem = {
    fault: 'cardWithoutAStencil',
    card: 1,
    face: null,
    field: '',
    text: 'a card under no stencil',
  }

  it('stands against the card it was read against', async () => {
    const { tab } = await open({ problems: [stencilless] })

    const marks = tab.marks()
    const second = tab.deck().cards[1]?.id ?? ''
    expect(marks.at.get(second)).toStrictEqual(['a card under no stencil'])
  })

  it('stands against no other card', async () => {
    const { tab } = await open({ problems: [stencilless] })

    const first = tab.deck().cards[0]?.id ?? ''
    expect(tab.marks().at.has(first)).toBe(false)
  })

  it('is nothing at all where the vault reported none', async () => {
    const { tab } = await open()

    expect(tab.marks().at.size).toBe(0)
    expect(tab.marks().whole).toStrictEqual([])
  })
})

describe('a value written over', () => {
  // One stencil named three ways: by a path, by a name carrying an alias, and
  // by the identifier of the note.
  const WRITTEN = ['cards/Animal', 'Animal|животное', 'note://01J3ZQ8W0T7K9V2M4N6P8R0S1T']

  /** A deck of one card, under the wikilink it wrote for its stencil. */
  const only = (stencil: string, stencilAt = 'Animal.md'): readonly Carded[] => [
    {
      mark: 'k7m2xq9fzp',
      section: null,
      heading: 'Llama',
      stencil,
      stencilAt,
      lead: '',
      values: [{ field: 'Name', text: 'Llama' }],
    },
  ]

  /** A card writing one field twice, which the grid draws two boxes. */
  const twice: readonly Carded[] = [
    {
      mark: 'k7m2xq9fzp',
      section: null,
      heading: 'Llama',
      stencil: 'Animal',
      stencilAt: 'Animal.md',
      lead: '',
      values: [
        { field: 'Name', text: 'Llama' },
        { field: 'Name', text: 'Alpaca' },
      ],
    },
  ]

  it('stands under the first field, whatever the card wrote in its brackets', async () => {
    for (const one of WRITTEN) {
      const { tab } = await open({ cards: only(one) })

      tab.writes('k7m2xq9fzp', 'Name', 1, 'Vicuña')

      expect(tab.deck().cards[0]?.values).toStrictEqual([{ field: 'Name', text: 'Vicuña' }])
    }
  })

  it('is written after the rest where the field is another one the stencil declares', async () => {
    const { tab } = await open({ cards: only('cards/Animal') })

    tab.writes('k7m2xq9fzp', 'Height', 1, 'about 45"')

    expect(tab.deck().cards[0]?.values).toStrictEqual([
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 45"' },
    ])
  })

  it('is the one of two under a field that was typed in, and the other stands', async () => {
    const { tab } = await open({ cards: twice })

    tab.writes('k7m2xq9fzp', 'Name', 2, 'Vicuña')

    expect(tab.deck().cards[0]?.values).toStrictEqual([
      { field: 'Name', text: 'Llama' },
      { field: 'Name', text: 'Vicuña' },
    ])
  })

  it('is written for a card whose link reached no stencil', async () => {
    const { tab } = await open({ cards: only('Gone', '') })

    tab.writes('k7m2xq9fzp', 'Name', 1, 'Vicuña')

    expect(tab.deck().cards[0]?.values).toStrictEqual([{ field: 'Name', text: 'Vicuña' }])
  })

  it('leaves the heading the file gave the card exactly as it stands', async () => {
    const { tab } = await open({ cards: only('Gone', '') })

    tab.writes('k7m2xq9fzp', 'Name', 1, 'Vicuña')

    expect(tab.deck().cards[0]?.heading).toBe('Llama')
  })
})

describe('a section of a deck the window holds', () => {
  it('is made at the end, and reaches the vault with the deck', async () => {
    const { decks, tab, wrote } = await open()

    tab.addsSection('Roots')
    expect(tab.bands().map((band) => band.name)).toStrictEqual(['Roots'])

    await decks.flush()
    expect(wrote()[0]?.sections).toStrictEqual([{ name: 'Roots', lead: '' }])
  })

  it('takes the name it was given', async () => {
    const { tab } = await open({ sections: [{ name: 'Roots', lead: '' }] })

    tab.namesSection(tab.bands()[0]?.id ?? '', 'Roots and shoots')

    expect(tab.bands().map((band) => band.name)).toStrictEqual(['Roots and shoots'])
  })

  it('takes away its heading and nothing else when it goes', async () => {
    const { tab } = await open({
      sections: [{ name: 'Roots', lead: '' }],
      cards: CARDS.map((card) => ({ ...card, section: 0 })),
    })

    tab.removesSection(tab.bands()[0]?.id ?? '')

    expect(tab.bands()).toStrictEqual([])
    expect(tab.deck().cards.map(calling)).toStrictEqual(['Llama', 'Alpaca'])
    expect(tab.drawn().map((card) => card.section)).toStrictEqual([null, null])
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

  // The file is read wrong in a way that stands against one card, so a mark
  // that went is a mark this can see going.
  const stencilless: Problem = {
    fault: 'cardWithoutAStencil',
    card: 1,
    face: null,
    field: '',
    text: 'a card under no stencil',
  }

  it('leaves what the grid is drawing standing, where the file reads the same', async () => {
    const one = await open({ problems: [stencilless] })
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

  it('leaves each mark standing on the card the grid is drawing', async () => {
    const one = await open({ problems: [stencilless] })
    // The card the grid is drawing, under the identity it was drawn with.
    const second = one.tab.deck().cards[1]?.id ?? ''

    one.decks.changed(['Animals.md'])
    await settles()

    expect(one.tab.marks().at.get(second)).toStrictEqual(['a card under no stencil'])
  })

  /* A card the file could not name was drawn under an identity the window
     minted. The reading that names it is that same card, so the tile it is
     being typed into stands, and the caret with it. */
  it('keeps the identity of a card the file has named since it was drawn', async () => {
    const one = await open()
    one.tab.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    const made = one.tab.deck().cards.at(-1)?.id ?? ''
    await one.decks.kept.settles(one.tab.id)

    // The deck was made whole where it was written: the card carries the mark
    // the core minted and the heading it read from the first field.
    one.holds([
      ...CARDS,
      {
        mark: 'w9s5jd2b1k',
        section: null,
        heading: 'Vicuña',
        stencil: 'Animal',
        stencilAt: 'Animal.md',
        lead: '',
        values: [{ field: 'Name', text: 'Vicuña' }],
      },
    ])
    one.decks.changed(['Animals.md'])
    await settles()

    expect(one.tab.deck().cards.at(-1)?.id).toBe(made)
    expect(one.tab.deck().cards.at(-1)?.mark).toBe('w9s5jd2b1k')
    expect(one.tab.drawn().at(-1)?.id).toBe(made)
  })

  /* A section carries no mark, so every reading mints one. The grid draws a run
     under the identity its section stands at, so a reading that mints fresh
     ones takes down every run and every card standing in it. */
  it('keeps the identity of a section the file still holds', async () => {
    const one = await open({ sections: [{ name: 'Roots', lead: '' }] })
    const stood = one.tab.deck().sections[0]?.id ?? ''
    one.tab.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await one.decks.kept.settles(one.tab.id)

    one.holds([
      ...CARDS,
      {
        mark: 'w9s5jd2b1k',
        section: null,
        heading: 'Vicuña',
        stencil: 'Animal',
        stencilAt: 'Animal.md',
        lead: '',
        values: [{ field: 'Name', text: 'Vicuña' }],
      },
    ])
    one.decks.changed(['Animals.md'])
    await settles()

    expect(one.tab.deck().sections[0]?.id).toBe(stood)
  })

  /* A heading is read back off a card's first field wherever the deck is
     written, so a deck reading as the one on screen leaves it standing. What
     the file says the heading is, though, is what the deck carries: nothing
     draws it and nothing is typed into it, so taking it moves nothing under
     the hand and the next write does not put the old one back. */
  it('takes a heading the file carries while the deck on screen stands', async () => {
    const one = await open()
    const stood = one.tab.deck().cards[0]?.id ?? ''

    one.holds(CARDS.map((card, at) => (at === 0 ? { ...card, heading: 'Llama and yak' } : card)))
    one.decks.changed(['Animals.md'])
    await settles()

    expect(one.tab.deck().cards[0]?.heading).toBe('Llama and yak')
    expect(one.tab.deck().cards[0]?.id).toBe(stood)
  })

  /* A section stands whatever the cards do. A card written into the file from
     somewhere else is a reading of a different length, and the sections of it
     are the sections that were already drawn. */
  it('keeps it where the file gained a card the window did not write', async () => {
    const one = await open({ sections: [{ name: 'Roots', lead: '' }] })
    const stood = one.tab.deck().sections[0]?.id ?? ''

    one.holds([
      ...CARDS,
      {
        mark: 'w9s5jd2b1k',
        section: null,
        heading: 'Vicuña',
        stencil: 'Animal',
        stencilAt: 'Animal.md',
        lead: '',
        values: [{ field: 'Name', text: 'Vicuña' }],
      },
    ])
    one.decks.changed(['Animals.md'])
    await settles()

    expect(one.tab.deck().cards).toHaveLength(CARDS.length + 1)
    expect(one.tab.deck().sections[0]?.id).toBe(stood)
  })

  /* A write takes a moment, and a person typing does not stop for it. The card
     that comes back named carries what it held when it was sent, and what is
     being typed into is the same card. */
  it('keeps it where the typing went on while the write was away', async () => {
    const one = await open()
    one.tab.adds('Animal', [{ field: 'Name', text: 'Vic' }], null)
    const made = one.tab.deck().cards.at(-1)?.id ?? ''
    await one.decks.kept.settles(one.tab.id)

    one.tab.writes(made, 'Name', 1, 'Vicuña')

    one.holds([
      ...CARDS,
      {
        mark: 'w9s5jd2b1k',
        section: null,
        heading: 'Vic',
        stencil: 'Animal',
        stencilAt: 'Animal.md',
        lead: '',
        values: [{ field: 'Name', text: 'Vic' }],
      },
    ])
    one.decks.changed(['Animals.md'])
    await settles()

    expect(one.tab.deck().cards.at(-1)?.id).toBe(made)
  })

  it('draws the file again where it was written from somewhere else', async () => {
    const one = await open()
    const was = drawing(one.tab)

    one.holds([
      {
        mark: 'w9s5jd2b1k',
        section: null,
        heading: 'Vicuña',
        stencil: 'Animal',
        stencilAt: 'Animal.md',
        lead: '',
        values: [{ field: 'Name', text: 'Vicuña' }],
      },
    ])
    one.decks.changed(['Animals.md'])
    await settles()

    expect(one.tab.drawn().map((card) => card.id)).toStrictEqual(['w9s5jd2b1k'])
    expect(one.tab.deck()).not.toBe(was.deck)
  })
})

describe('a deck renamed under the window', () => {
  /** The file the deck was opened at, under the name it was given instead. */
  const renamed = (one: Awaited<ReturnType<typeof open>>) =>
    one.decks.changed(['Beasts.md'], [{ from: 'Animals.md', to: 'Beasts.md' }])

  it('is the tab it has when it is asked for at the name it now carries', async () => {
    const one = await open()

    renamed(one)
    await settles()
    one.road.made('Beasts.md', '', 'deck')
    await settles()

    expect(one.decks.all()).toHaveLength(1)
    expect(one.held.host.each(DECK)).toHaveLength(1)
  })

  it('keeps what the vault said about it, which stands at the name it went to', async () => {
    const one = await open({
      problems: [
        {
          fault: 'cardWithoutAStencil',
          card: 1,
          face: null,
          field: '',
          text: 'a card under no stencil',
        },
      ],
    })

    renamed(one)

    // Before the read of the file under its new name has answered.
    expect(one.tab.marks().at.size).toBe(1)
  })

  it('is called what the file was called, before the name it went to is read', async () => {
    const one = await open()

    renamed(one)

    expect(one.decks.called('Beasts.md')).toBe('Animals')
  })
})

describe('a deck whose tab has gone', () => {
  it('is nothing the window still says a word about', async () => {
    const one = await open()
    expect(one.decks.called('Animals.md')).toBe('Animals')

    one.held.shut(one.id)
    await settles()

    expect(one.decks.called('Animals.md')).toBe('Animals.md')
  })
})

describe('a deck the vault could not be reached for', () => {
  it('says the vault could not be reached, where the read reached nothing', async () => {
    const { tab } = await open({ unreachable: true })

    expect(tab.saying()).toBe(words.unreachable)
  })

  it('says the file could not be written, where that is what was refused', async () => {
    const { decks, tab } = await open({ wrote: 'unreadable' })

    tab.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.kept.settles(decks.all()[0] ?? '')

    expect(tab.saying()).toBe(words.notSaved)
  })
})

describe('the stencils a listing did not answer with', () => {
  it('are asked for again when the deck comes back on screen', async () => {
    const one = await open({ unlisted: 1 })
    expect(one.tab.cuts()).toStrictEqual([])

    one.decks.kind.shown?.(one.tab, one.id)
    await settles()

    expect(one.tab.cuts()).toStrictEqual([{ name: 'Animal', fields: ['Name', 'Height'] }])
  })
})

