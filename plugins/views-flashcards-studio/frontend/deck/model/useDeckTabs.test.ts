/**
 * What a deck tab holds, asked without a screen: what it reads, what it writes
 * back, and what it does when the file moved past what it read.
 */
import { describe, expect, it } from 'vitest'
import { StopReason } from '../../api/types.js'
import { asFailure, asValue } from '@numen/plugin-sdk'
import type { ErrorCode } from '../../cards/wire/errors'
import type { Cards, VaultCard, DeckProblem, Preset } from '../../cards'
import { DEFAULTS, NO_CURVE_POINT, NO_BOUNDS, type PresetChoice, type Presets } from '../../cards'
import { createMockBufferHost, createMockTabWindow } from '@numen/plugin-sdk/testing'
import { DECK } from '../kind'
import { useDeckTabs as useDeckTabsOriginal, type DeckTabState } from './useDeckTabs'
import { WORDS as words } from '../../cards'

/** The deck tabs of a window that holds them in memory, and the buffers they are written out through. */
const useDeckTabs = (cards: Cards, presets: Presets) => {
  const windowTabs = createMockTabWindow()
  const buffers = createMockBufferHost()
  const decks = useDeckTabsOriginal(cards, presets, { buffers, closeTab: windowTabs.finishClose })
  windowTabs.register(decks.kind)
  return { windowTabs, buffers, decks }
}

/** A preset that schedules, which is what every preset here is. */
const SCHEDULING = { stops: StopReason.NOTHING, stopsOn: StopReason.NOTHING } satisfies Pick<
  Preset,
  'stops' | 'stopsOn'
>

/** A moment for whatever the tab asked the vault for to come back. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

// The two cards name one card model two ways: by a path from the root, and by a
// name carrying an alias. Both reach the file the vault says they reach.
const CARDS: readonly VaultCard[] = [
  {
    cardId: 'k7m2xq9fzp',
    sectionIndex: null,
    heading: 'Llama',
    cardModelLink: 'cards/Animal',
    cardModelPath: 'Animal.md',
    preamble: '',
    values: [
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 45"' },
    ],
  },
  {
    cardId: '3n8vr4tqch',
    sectionIndex: null,
    heading: 'Alpaca',
    cardModelLink: 'Animal|животное',
    cardModelPath: 'Animal.md',
    preamble: '',
    values: [{ field: 'Name', text: 'Alpaca' }],
  },
]

/** What a card is called on a screen: the first line of its first field. */
const getCardName = (card: { values: readonly { field: string; text: string }[] }): string =>
  card.values[0]?.text || '—'

/** A vault holding one deck, writing down every write it was asked for. */
const vault = (
  answers: {
    error?: ErrorCode | 'changed'
    problems?: readonly DeckProblem[]
    bound?: number
    /** The write answers that the file moved past what the tab read. */
    isChanged?: boolean
    /** The cards the file holds, where a test wants other ones. */
    cards?: readonly VaultCard[]
    /** The sections the file holds, where a test wants some. */
    sections?: readonly { name: string; preamble: string }[]
    /** The vault is out of reach, and a read of the deck reaches nothing. */
    isUnreachable?: boolean
    /** What a write of the deck fails for. */
    wrote?: ErrorCode
    /** How many listings of the card models go unanswered before one answers. */
    unlisted?: number
    /** The presets the vault holds, where a test wants other ones. */
    presets?: readonly PresetChoice[]
    /** The preset the deck names, and nothing for a deck naming none. */
    by?: string
    /** What is said against what the deck names. */
    saying?: string
    /** What putting the deck on a preset fails for. */
    notScheduled?: ErrorCode
    /** Putting the deck on a preset answers that the file moved past it. */
    hasSchedulingChanged?: boolean
  } = {},
) => {
  const content: string[] = []
  /** Each deck as the vault was handed it, for what a card carries to be asked. */
  const wrote: Parameters<Cards['writeDeck']>[1][] = []
  const observed: (string | null)[] = []
  let reads = 0
  let cards: readonly VaultCard[] = answers.cards ?? CARDS

  let items = 0

  const core: Cards = {
    cardModels: async () => {
      items += 1
      if (items <= (answers.unlisted ?? 0)) throw new Error('out of reach')
      return {
        cardModels: [{ path: 'Animal.md', title: 'Animal', fields: ['Name', 'Height'] }],
        held: 1,
      }
    },
    createDeck: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    createCardModel: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    renameField: async () => asValue({ decks: [], cards: 0, notWritten: [], at: '' }),
    readDeck: async (path) => {
      reads += 1
      if (answers.isUnreachable) throw new Error('out of reach')
      if (answers.error) {
        return asFailure({ code: answers.error, bound: answers.bound ?? 0 })
      }
      return asValue({
        deck: {
          path,
          title: 'Animals',
          preamble: '',
          cards,
          sections: answers.sections ?? [],
          tail: '',
          problems: answers.problems ?? [],
        },
        at: `read ${reads}`,
      })
    },
    writeDeck: async (path, deck, presented) => {
      content.push(`${path} ${deck.cards.map(getCardName).join(', ') || '—'}`)
      wrote.push(deck)
      observed.push(presented)
      if (answers.wrote) return asFailure({ code: answers.wrote, bound: 0 })
      if (answers.isChanged) return asFailure({ code: 'changed' as const, bound: 0 })
      cards = deck.cards
      return asValue({ at: 'written' })
    },
    readCardModel: async () => asFailure('missing' as const),
    writeCardModel: async () => asValue({ at: '' }),
  }

  /** Which preset the deck names, as the vault answers it. */
  let presetPath = answers.by ?? ''
  /** Every deck put on a preset, as the window asked for it. */
  const attached: { deck: string; preset: string; seen: string }[] = []

  const presets: Presets = {
    read: async (path) =>
      asValue({
        preset: { path, title: 'Sanskrit', settings: DEFAULTS, problems: [], ...SCHEDULING },
        at: '',
        bounds: NO_BOUNDS,
      }),
    list: async () =>
      answers.presets ?? [
        { path: 'Sanskrit.md', title: 'Sanskrit' },
        { path: 'presets/Slow.md', title: '' },
      ],
    createPreset: async () => asValue({ path: '' }),
    getDeckPreset: async () =>
      asValue({
        preset: {
          path: presetPath,
          title: presetPath === 'Sanskrit.md' ? 'Sanskrit' : '',
          settings: DEFAULTS,
          problems: answers.saying ? [answers.saying] : [],
          ...SCHEDULING,
        },
        at: '',
        bounds: NO_BOUNDS,
      }),
    scheduleDeck: async (deck, preset, observed) => {
      attached.push({ deck, preset, seen: observed })
      if (answers.notScheduled) return asFailure(answers.notScheduled)
      if (answers.hasSchedulingChanged) return asFailure('changed' as const)
      presetPath = preset
      return asValue({ at: 'scheduled' })
    },
    write: async () => asValue({ at: '' }),
    curve: async () => ({
      goal: 'minutes',
      grid: [],
      days: [],
      at: [],
      now: NO_CURVE_POINT,
      suggested: NO_CURVE_POINT,
      decks: 0,
      cards: 0,
      overdue: 0,
      unbegun: 0,
      isHonest: true,
    }),
  }

  return {
    core,
    presets,
    attached,
    written: content,
    wrote: () => wrote,
    seen: observed,
    observed,
    getReadCount: () => reads,
    getListCount: () => items,
    /** The file written from somewhere else, which the next read answers with. */
    setCards: (next: readonly VaultCard[]) => {
      cards = next
    },
  }
}

/** A window with one deck open on a file, and what that tab holds. */
const open = async (answers: Parameters<typeof vault>[0] = {}, path = 'Animals.md') => {
  const one = vault(answers)
  const { windowTabs, buffers, decks } = useDeckTabs(one.core, one.presets)
  const id = await windowTabs.openTab(DECK, path)
  await settle()
  const tab = windowTabs.getTabState<DeckTabState>(id) as DeckTabState
  return { ...one, windowTabs, buffers, decks, id, tab }
}

describe('a deck opened', () => {
  it('draws the cards the vault read, in the order the file had them', async () => {
    const { tab } = await open()

    expect(tab.deck.value.cards.map(getCardName)).toStrictEqual(['Llama', 'Alpaca'])
  })

  it('knows each card by the card id the file carries for it', async () => {
    const { tab } = await open()

    expect(tab.deck.value.cards.map((card) => card.id)).toStrictEqual(['k7m2xq9fzp', '3n8vr4tqch'])
    expect(tab.deckCards.value.map((card) => card.id)).toStrictEqual(['k7m2xq9fzp', '3n8vr4tqch'])
  })

  it('draws the sections the vault read, each under an identity of its own', async () => {
    const { tab } = await open({ sections: [{ name: 'Roots', preamble: '' }] })

    expect(tab.sections.value.map((section) => section.name)).toStrictEqual(['Roots'])
    expect(tab.sections.value[0]?.id).toBeTruthy()
  })

  it('is called what the file is called', async () => {
    const { decks, id, windowTabs } = await open()

    expect(decks.kind.getTitle?.(windowTabs.getTabState<DeckTabState>(id) as DeckTabState)).toBe(
      'Animals',
    )
  })

  it('offers every card model the vault holds as a cut', async () => {
    const { tab } = await open()

    expect(tab.cardModels.value).toStrictEqual([{ name: 'Animal', fields: ['Name', 'Height'] }])
  })

  it('draws each card under the card model its wikilink reached', async () => {
    const { tab } = await open()

    expect(tab.deckCards.value.map((card) => card.cardModel)).toStrictEqual(['Animal', 'Animal'])
  })

  it('is one tab per file, so the same deck requested for twice is the tab it has', async () => {
    const { windowTabs, id } = await open()

    const again = await windowTabs.openTab(DECK, 'Animals.md')

    expect(again).toBe(id)
  })

  it('carries no mark while nothing is unwritten', async () => {
    const { decks, tab } = await open()

    expect(decks.kind.getMark?.(tab)).toBeUndefined()
  })
})

describe('a card written in a deck', () => {
  it('remains in the deck the tab draws', async () => {
    const { tab } = await open()

    tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)

    expect(tab.deck.value.cards.map(getCardName)).toStrictEqual(['Llama', 'Alpaca', 'Vicuña'])
  })

  it('is placed under where the vault files the card model it was cut by', async () => {
    const { tab } = await open()

    tab.addCard('Animal', [], null)

    expect(tab.deck.value.cards.at(-1)?.cardModelPath).toBe('Animal.md')
  })

  it('carries no card id, which is written where the deck is made whole', async () => {
    const { tab } = await open()

    tab.addCard('Animal', [], null)

    expect(tab.deck.value.cards.at(-1)?.cardId).toBe('')
  })

  it('leaves the tab unsaved, and marks it', async () => {
    const { decks, tab } = await open()

    tab.addCard('Animal', [], null)

    expect(tab.note.value.state).toBe('unsaved')
    expect(decks.kind.getMark?.(tab)).toBe('unsaved')
  })

  it('reaches the vault as the cards of that file, with the file it was read from', async () => {
    const { decks, tab, written: savedFiles, observed: observedRanks } = await open()

    tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.store.flush()

    expect(savedFiles).toStrictEqual(['Animals.md Llama, Alpaca, Vicuña'])
    expect(observedRanks).toStrictEqual(['read 1'])
  })

  it('reaches the vault carrying the card id and the heading the file gave it', async () => {
    const { decks, tab, wrote } = await open()

    tab.writeCardField('k7m2xq9fzp', 'Height', 1, 'about 46"')
    await decks.store.flush()

    expect(wrote()[0]?.cards[0]?.cardId).toBe('k7m2xq9fzp')
    expect(wrote()[0]?.cards[0]?.heading).toBe('Llama')
  })

  it('writes nothing where nothing was touched', async () => {
    const { decks, written: savedFiles } = await open()

    await decks.store.flush()

    expect(savedFiles).toStrictEqual([])
  })
})

describe('a deck whose file moved past what was read', () => {
  it('is stale once the write arrives back saying the file changed', async () => {
    const { decks, tab } = await open({ isChanged: true })

    tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.store.flush()

    expect(tab.note.value.state).toBe('stale')
  })

  it('keeps what the person wrote when they say so, over whatever the file holds', async () => {
    const {
      decks,
      tab,
      written: savedFiles,
      observed: observedRanks,
    } = await open({ isChanged: true })

    tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.store.flush()
    tab.keepMine()
    await settle()

    expect(savedFiles).toHaveLength(2)
    expect(observedRanks[1]).toBeNull()
  })

  it('reads the file again when the person takes what it holds', async () => {
    const { decks, tab, getReadCount } = await open({ isChanged: true })

    tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.store.flush()
    tab.takeFile()
    await settle()

    expect(getReadCount()).toBe(2)
    expect(tab.note.value.state).toBe('clean')
    expect(tab.deck.value.cards.map(getCardName)).toStrictEqual(['Llama', 'Alpaca'])
  })
})

describe('a deck the vault rejected', () => {
  it('says which bound it is over, and draws no card', async () => {
    const { tab } = await open({ error: 'deckTooLarge', bound: 8388608 })

    expect(tab.errorMessage.value).toBe(words.tooLarge(8388608))
    expect(tab.deck.value.cards).toStrictEqual([])
  })

  it('says the note is not a deck where that is what it is', async () => {
    const { tab } = await open({ error: 'notADeck' })

    expect(tab.errorMessage.value).toBe(words.notADeck)
  })

  it('says nothing where the deck was read', async () => {
    const { tab } = await open()

    expect(tab.errorMessage.value).toBe('')
  })
})

describe('what is wrong with a deck', () => {
  const withoutCardModel: DeckProblem = {
    kind: 'cardWithoutACardModel',
    card: 1,
    template: null,
    field: '',
    text: 'a card under no card model',
  }

  it('remains against the card it was read against', async () => {
    const { tab } = await open({ problems: [withoutCardModel] })

    const marks = tab.marks.value
    const second = tab.deck.value.cards[1]?.id ?? ''
    expect(marks.at.get(second)).toStrictEqual(['a card under no card model'])
  })

  it('remains against no other card', async () => {
    const { tab } = await open({ problems: [withoutCardModel] })

    const first = tab.deck.value.cards[0]?.id ?? ''
    expect(tab.marks.value.at.has(first)).toBe(false)
  })

  it('is nothing at all where the vault reported none', async () => {
    const { tab } = await open()

    expect(tab.marks.value.at.size).toBe(0)
    expect(tab.marks.value.whole).toStrictEqual([])
  })
})

describe('a value written over', () => {
  // One card model named three ways: by a path, by a name carrying an alias, and
  // by the identifier of the note.
  const WRITTEN = ['cards/Animal', 'Animal|животное', 'note://01J3ZQ8W0T7K9V2M4N6P8R0S1T']

  /** A deck of one card, under the wikilink it wrote for its card model. */
  const createOneCard = (cardModel: string, cardModelPath = 'Animal.md'): readonly VaultCard[] => [
    {
      cardId: 'k7m2xq9fzp',
      sectionIndex: null,
      heading: 'Llama',
      cardModelLink: cardModel,
      cardModelPath,
      preamble: '',
      values: [{ field: 'Name', text: 'Llama' }],
    },
  ]

  /** A card writing one field twice, which the grid draws two boxes. */
  const twice: readonly VaultCard[] = [
    {
      cardId: 'k7m2xq9fzp',
      sectionIndex: null,
      heading: 'Llama',
      cardModelLink: 'Animal',
      cardModelPath: 'Animal.md',
      preamble: '',
      values: [
        { field: 'Name', text: 'Llama' },
        { field: 'Name', text: 'Alpaca' },
      ],
    },
  ]

  it('is placed under the first field, whatever the card wrote in its brackets', async () => {
    for (const one of WRITTEN) {
      const { tab } = await open({ cards: createOneCard(one) })

      tab.writeCardField('k7m2xq9fzp', 'Name', 1, 'Vicuña')

      expect(tab.deck.value.cards[0]?.values).toStrictEqual([{ field: 'Name', text: 'Vicuña' }])
    }
  })

  it('is written after the rest where the field is another one the card model declares', async () => {
    const { tab } = await open({ cards: createOneCard('cards/Animal') })

    tab.writeCardField('k7m2xq9fzp', 'Height', 1, 'about 45"')

    expect(tab.deck.value.cards[0]?.values).toStrictEqual([
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 45"' },
    ])
  })

  it('is the one of two under a field that was typed in, and the other remains', async () => {
    const { tab } = await open({ cards: twice })

    tab.writeCardField('k7m2xq9fzp', 'Name', 2, 'Vicuña')

    expect(tab.deck.value.cards[0]?.values).toStrictEqual([
      { field: 'Name', text: 'Llama' },
      { field: 'Name', text: 'Vicuña' },
    ])
  })

  it('is written for a card whose link reached no card model', async () => {
    const { tab } = await open({ cards: createOneCard('Gone', '') })

    tab.writeCardField('k7m2xq9fzp', 'Name', 1, 'Vicuña')

    expect(tab.deck.value.cards[0]?.values).toStrictEqual([{ field: 'Name', text: 'Vicuña' }])
  })

  it('leaves the heading the file gave the card exactly currently', async () => {
    const { tab } = await open({ cards: createOneCard('Gone', '') })

    tab.writeCardField('k7m2xq9fzp', 'Name', 1, 'Vicuña')

    expect(tab.deck.value.cards[0]?.heading).toBe('Llama')
  })
})

describe('a section of a deck the window holds', () => {
  it('is made at the end, and reaches the vault with the deck', async () => {
    const { decks, tab, wrote } = await open()

    tab.addSection('Roots')
    expect(tab.sections.value.map((section) => section.name)).toStrictEqual(['Roots'])

    await decks.store.flush()
    expect(wrote()[0]?.sections).toStrictEqual([{ name: 'Roots', preamble: '' }])
  })

  it('takes the name it was given', async () => {
    const { tab } = await open({ sections: [{ name: 'Roots', preamble: '' }] })

    tab.renameSection(tab.sections.value[0]?.id ?? '', 'Roots and shoots')

    expect(tab.sections.value.map((section) => section.name)).toStrictEqual(['Roots and shoots'])
  })

  it('takes away its heading and nothing else when it goes', async () => {
    const { tab } = await open({
      sections: [{ name: 'Roots', preamble: '' }],
      cards: CARDS.map((card) => ({ ...card, sectionIndex: 0 })),
    })

    expect(tab.deckCards.value.map((card) => card.section)).toStrictEqual([
      tab.sections.value[0]?.id,
      tab.sections.value[0]?.id,
    ])

    tab.removeSection(tab.sections.value[0]?.id ?? '')

    expect(tab.sections.value).toStrictEqual([])
    expect(tab.deck.value.cards.map(getCardName)).toStrictEqual(['Llama', 'Alpaca'])
    expect(tab.deckCards.value.map((card) => card.section)).toStrictEqual([null, null])
  })
})

describe('the vault changing under the window', () => {
  it('asks for no card model while the window holds no deck', async () => {
    const one = vault()
    const { decks } = useDeckTabs(one.core, one.presets)

    decks.applyPathChanges(['Notes.md'])
    await settle()

    expect(one.getListCount()).toBe(0)
  })

  it('lists them again once a deck is open', async () => {
    const { decks, getListCount: listed } = await open()

    decks.applyPathChanges(['Notes.md'])
    await settle()

    expect(listed()).toBe(2)
  })
})

describe('a deck read again under the window', () => {
  /** Everything the tab hands the grid to draw. */
  const getTabSnapshot = (tab: DeckTabState) => ({
    deck: tab.deck.value,
    deckCards: tab.deckCards.value,
    cardModels: tab.cardModels.value,
    marks: tab.marks.value,
  })

  // The file is read wrong in a way that stands against one card, so a mark
  // that went is a mark this can see going.
  const withoutCardModel: DeckProblem = {
    kind: 'cardWithoutACardModel',
    card: 1,
    template: null,
    field: '',
    text: 'a card under no card model',
  }

  it('leaves what the grid is drawing standing, where the file reads the same', async () => {
    const one = await open({ problems: [withoutCardModel] })
    const was = getTabSnapshot(one.tab)

    one.decks.applyPathChanges(['Animals.md'])
    await settle()

    // Each of them the same thing, and not merely a thing that reads the same:
    // a card under the keyboard is redrawn by anything else.
    expect(getTabSnapshot(one.tab)).toStrictEqual(was)
    expect(one.tab.deck.value).toBe(was.deck)
    expect(one.tab.deckCards.value).toBe(was.deckCards)
    expect(one.tab.cardModels.value).toBe(was.cardModels)
    expect(one.tab.marks.value).toBe(was.marks)
  })

  it('leaves each mark standing on the card the grid is drawing', async () => {
    const one = await open({ problems: [withoutCardModel] })
    // The card the grid is drawing, under the identity it was drawn with.
    const second = one.tab.deck.value.cards[1]?.id ?? ''

    one.decks.applyPathChanges(['Animals.md'])
    await settle()

    expect(one.tab.marks.value.at.get(second)).toStrictEqual(['a card under no card model'])
  })

  /* A card the file could not name was drawn under an identity the window
     minted. The reading that names it is that same card, so the tile it is
     being typed into stands, and the caret with it. */
  it('keeps the identity of a card the file has named since it was drawn', async () => {
    const one = await open()
    one.tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    const createdItem = one.tab.deck.value.cards.at(-1)?.id ?? ''
    await one.decks.store.flushPending(one.tab.id)

    // The deck was made whole where it was written: the card carries the card id
    // the core minted and the heading it read from the first field.
    one.setCards([
      ...CARDS,
      {
        cardId: 'w9s5jd2b1k',
        sectionIndex: null,
        heading: 'Vicuña',
        cardModelLink: 'Animal',
        cardModelPath: 'Animal.md',
        preamble: '',
        values: [{ field: 'Name', text: 'Vicuña' }],
      },
    ])
    one.decks.applyPathChanges(['Animals.md'])
    await settle()

    expect(one.tab.deck.value.cards.at(-1)?.id).toBe(createdItem)
    expect(one.tab.deck.value.cards.at(-1)?.cardId).toBe('w9s5jd2b1k')
    expect(one.tab.deckCards.value.at(-1)?.id).toBe(createdItem)
  })

  it('keeps the identity of a section the file still holds', async () => {
    const one = await open({ sections: [{ name: 'Roots', preamble: '' }] })
    const status = one.tab.deck.value.sections[0]?.id ?? ''
    one.tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await one.decks.store.flushPending(one.tab.id)

    one.setCards([
      ...CARDS,
      {
        cardId: 'w9s5jd2b1k',
        sectionIndex: null,
        heading: 'Vicuña',
        cardModelLink: 'Animal',
        cardModelPath: 'Animal.md',
        preamble: '',
        values: [{ field: 'Name', text: 'Vicuña' }],
      },
    ])
    one.decks.applyPathChanges(['Animals.md'])
    await settle()

    expect(one.tab.deck.value.sections[0]?.id).toBe(status)
  })

  /* A heading is read back off a card's first field wherever the deck is
     written, so a deck reading as the one on screen leaves it standing. What
     the file says the heading is, though, is what the deck carries: nothing
     draws it and nothing is typed into it, so taking it moves nothing under
     the hand and the next write does not put the old one back. */
  it('takes a heading the file carries while the deck on screen remains', async () => {
    const one = await open()
    const status = one.tab.deck.value.cards[0]?.id ?? ''

    one.setCards(
      CARDS.map((card, index) => (index === 0 ? { ...card, heading: 'Llama and yak' } : card)),
    )
    one.decks.applyPathChanges(['Animals.md'])
    await settle()

    expect(one.tab.deck.value.cards[0]?.heading).toBe('Llama and yak')
    expect(one.tab.deck.value.cards[0]?.id).toBe(status)
  })

  /* A section stands whatever the cards do. A card written into the file from
     somewhere else is a reading of a different length, and the sections of it
     are the sections that were already drawn. */
  it('keeps it where the file gained a card the window did not write', async () => {
    const one = await open({ sections: [{ name: 'Roots', preamble: '' }] })
    const status = one.tab.deck.value.sections[0]?.id ?? ''

    one.setCards([
      ...CARDS,
      {
        cardId: 'w9s5jd2b1k',
        sectionIndex: null,
        heading: 'Vicuña',
        cardModelLink: 'Animal',
        cardModelPath: 'Animal.md',
        preamble: '',
        values: [{ field: 'Name', text: 'Vicuña' }],
      },
    ])
    one.decks.applyPathChanges(['Animals.md'])
    await settle()

    expect(one.tab.deck.value.cards).toHaveLength(CARDS.length + 1)
    expect(one.tab.deck.value.sections[0]?.id).toBe(status)
  })

  /* A write takes a moment, and a person typing does not stop for it. The card
     that comes back named carries what it windowTabs when it was sent, and what is
     being typed into is the same card. */
  it('keeps it where the typing went on while the write was away', async () => {
    const one = await open()
    one.tab.addCard('Animal', [{ field: 'Name', text: 'Vic' }], null)
    const createdItem = one.tab.deck.value.cards.at(-1)?.id ?? ''
    await one.decks.store.flushPending(one.tab.id)

    one.tab.writeCardField(createdItem, 'Name', 1, 'Vicuña')

    one.setCards([
      ...CARDS,
      {
        cardId: 'w9s5jd2b1k',
        sectionIndex: null,
        heading: 'Vic',
        cardModelLink: 'Animal',
        cardModelPath: 'Animal.md',
        preamble: '',
        values: [{ field: 'Name', text: 'Vic' }],
      },
    ])
    one.decks.applyPathChanges(['Animals.md'])
    await settle()

    expect(one.tab.deck.value.cards.at(-1)?.id).toBe(createdItem)
  })

  it('draws the file again where it was written from somewhere else', async () => {
    const one = await open()
    const was = getTabSnapshot(one.tab)

    one.setCards([
      {
        cardId: 'w9s5jd2b1k',
        sectionIndex: null,
        heading: 'Vicuña',
        cardModelLink: 'Animal',
        cardModelPath: 'Animal.md',
        preamble: '',
        values: [{ field: 'Name', text: 'Vicuña' }],
      },
    ])
    one.decks.applyPathChanges(['Animals.md'])
    await settle()

    expect(one.tab.deckCards.value.map((card) => card.id)).toStrictEqual(['w9s5jd2b1k'])
    expect(one.tab.deck.value).not.toBe(was.deck)
  })
})

describe('a deck renamed under the window', () => {
  /** The file the deck was opened at, under the name it was given instead. */
  const renameDeck = (one: Awaited<ReturnType<typeof open>>) =>
    one.decks.applyPathChanges(['Beasts.md'], [{ from: 'Animals.md', to: 'Beasts.md' }])

  it('is the tab it has when it is requested for at the name it now carries', async () => {
    const one = await open()

    renameDeck(one)
    await settle()
    await one.windowTabs.openTab(DECK, 'Beasts.md')
    await settle()

    expect(one.decks.store.getOpenIds()).toHaveLength(1)
    expect(one.windowTabs.each(DECK)).toHaveLength(1)
  })

  it('keeps what the vault returned about it, which remains at the name it went to', async () => {
    const one = await open({
      problems: [
        {
          kind: 'cardWithoutACardModel',
          card: 1,
          template: null,
          field: '',
          text: 'a card under no card model',
        },
      ],
    })

    renameDeck(one)

    // Before the read of the file under its new name has answered.
    expect(one.tab.marks.value.at.size).toBe(1)
  })

  it('is called what the file was called, before the name it went to is read', async () => {
    const one = await open()

    renameDeck(one)

    expect(one.decks.getTitle('Beasts.md')).toBe('Animals')
  })
})

describe('a deck whose tab has gone', () => {
  it('is nothing the window still says a word about', async () => {
    const one = await open()
    expect(one.decks.getTitle('Animals.md')).toBe('Animals')

    one.windowTabs.requestClose(one.id)
    await settle()

    expect(one.decks.getTitle('Animals.md')).toBe('Animals.md')
  })
})

describe('a deck the vault could not be reached for', () => {
  it('says the vault could not be reached, where the read reached nothing', async () => {
    const { tab } = await open({ isUnreachable: true })

    expect(tab.errorMessage.value).toBe(words.unreachable)
  })

  it('says the file could not be written, where that is what failed', async () => {
    const { decks, tab } = await open({ wrote: 'unreadable' })

    tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await decks.store.flushPending(decks.store.getOpenIds()[0] ?? '')

    expect(tab.errorMessage.value).toBe(words.notSaved)
  })
})

describe('the card models a listing did not answer with', () => {
  it('are requested for again when the deck arrives back on screen', async () => {
    const one = await open({ unlisted: 1 })
    expect(one.tab.cardModels.value).toStrictEqual([])

    one.decks.kind.onShow?.(one.tab, one.id)
    await settle()

    expect(one.tab.cardModels.value).toStrictEqual([{ name: 'Animal', fields: ['Name', 'Height'] }])
  })
})

describe('the preset a deck is scheduled by', () => {
  it('is what the vault says the deck names', async () => {
    const { tab } = await open({ by: 'Sanskrit.md' })

    expect(tab.scheduled.value).toStrictEqual({
      path: 'Sanskrit.md',
      name: 'Sanskrit',
      errorMessage: '',
    })
  })

  it('is the defaults for a deck naming none', async () => {
    const { tab } = await open()

    expect(tab.scheduled.value).toStrictEqual({ path: '', name: words.defaults, errorMessage: '' })
  })

  it('offers the defaults first, and a preset nothing names by its file', async () => {
    const { tab } = await open()

    expect(tab.choices.value).toStrictEqual([
      { path: '', name: words.defaults },
      { path: 'Sanskrit.md', name: 'Sanskrit' },
      { path: 'presets/Slow.md', name: 'Slow.md' },
    ])
  })

  it('carries the file the deck was read at, so a write lands on what was read', async () => {
    const { tab, attached } = await open()

    tab.setSchedule('Sanskrit.md')
    await settle()

    expect(attached).toStrictEqual([{ deck: 'Animals.md', preset: 'Sanskrit.md', seen: 'read 1' }])
  })

  it('writes what the deck owes before it writes the preset', async () => {
    const { tab, attached, written: savedFiles } = await open()
    tab.writeCardField('k7m2xq9fzp', 'Name', 1, 'Vicuña')

    tab.setSchedule('Sanskrit.md')
    await settle()

    expect(savedFiles).toStrictEqual(['Animals.md Vicuña, Alpaca'])
    expect(attached.map((one) => one.preset)).toStrictEqual(['Sanskrit.md'])
    // The deck was written first, so what the preset write presents is the file
    // that write made.
    expect(attached[0]?.seen).toBe('written')
  })

  it('reads the deck again, so the tab writes against the file the choice made', async () => {
    const { tab, getReadCount } = await open()
    const was = getReadCount()

    tab.setSchedule('Sanskrit.md')
    await settle()

    expect(getReadCount()).toBeGreaterThan(was)
    expect(tab.scheduled.value.path).toBe('Sanskrit.md')
  })

  it('says a choice the vault would not write, and leaves the preset standing', async () => {
    const { tab } = await open({ by: 'Sanskrit.md', notScheduled: 'notAPreset' })

    tab.setSchedule('presets/Slow.md')
    await settle()

    expect(tab.scheduled.value.errorMessage).toBe(words.notScheduled)
    expect(tab.scheduled.value.path).toBe('Sanskrit.md')
  })

  it('says a deck the file moved past since the window read it', async () => {
    const { tab } = await open({ hasSchedulingChanged: true })

    tab.setSchedule('Sanskrit.md')
    await settle()

    expect(tab.scheduled.value.errorMessage).toBe(words.notScheduledChanged)
    expect(tab.scheduled.value.path).toBe('')
  })

  it('says what the deck names and the vault does not hold', async () => {
    const { tab } = await open({ saying: 'Sanskrit reaches no note, and the defaults stand' })

    expect(tab.scheduled.value.path).toBe('')
    expect(tab.scheduled.value.errorMessage).toContain('reaches no note')
  })
})

describe('a deck the vault rejected', () => {
  it('draws no mark, since nothing was read to mark', async () => {
    const { tab } = await open({ error: 'notADeck' })

    expect(tab.marks.value.fields.size).toBe(0)
    expect(tab.marks.value.whole).toStrictEqual([])
  })

  it('fails under the code the vault gave', async () => {
    const { tab } = await open({ error: 'deckTooLarge', bound: 10 })

    expect(tab.note.value.error).toBe('tooLarge')
  })

  it('fails as not a deck where the code names no error', async () => {
    const { tab } = await open({ error: 'changed' })

    expect(tab.note.value.error).toBe('notANote')
  })
})

describe('a card standing in a deck that was turned', () => {
  it('is the card it was before the turn', async () => {
    const { tab } = await open()
    const llama = tab.deck.value.cards[0]

    tab.addCard('Animal', [], null)

    expect(tab.deck.value.cards[0]).toBe(llama)
  })
})

describe('a deck tab closed', () => {
  it('goes from the window, with the transcript about its file', async () => {
    const one = await open()

    one.tab.close(one.id)
    await settle()

    expect(one.windowTabs.each(DECK)).toHaveLength(0)
    expect(one.decks.getTitle('Animals.md')).toBe('Animals.md')
  })

  it('forgets the transcript about its file while a tab at another file stays', async () => {
    const one = await open()
    await one.windowTabs.openTab(DECK, 'Beasts.md')
    await settle()

    one.tab.close(one.id)
    await settle()

    expect(one.decks.getTitle('Animals.md')).toBe('Animals.md')
    expect(one.decks.getTitle('Beasts.md')).toBe('Animals')
  })

  it('stays where the file moved past what it holds', async () => {
    const one = await open({ isChanged: true })
    one.tab.addCard('Animal', [], null)
    await one.decks.store.flush()

    one.tab.close(one.id)
    await settle()

    expect(one.windowTabs.each(DECK)).toHaveLength(1)
    expect(one.decks.getTitle('Animals.md')).toBe('Animals')
  })

  it('keeps the transcript about a file another tab still remains at', async () => {
    const one = await open()
    await one.windowTabs.openTab(DECK, 'Beasts.md')
    await settle()
    one.decks.applyPathChanges(['Animals.md'], [{ from: 'Beasts.md', to: 'Animals.md' }])
    await settle()
    expect(one.decks.store.getPath('deck:Beasts.md')).toBe('Animals.md')

    one.tab.close(one.id)
    await settle()

    expect(one.decks.getTitle('Animals.md')).toBe('Animals')
  })

  it('forgets the preset its file was scheduled by', async () => {
    const one = await open({ by: 'Sanskrit.md' })
    expect(one.tab.scheduled.value.path).toBe('Sanskrit.md')
    one.tab.close(one.id)
    await settle()
    one.presets.getDeckPreset = () => new Promise(() => {})

    const again = await one.windowTabs.openTab(DECK, 'Animals.md')

    const state = one.windowTabs.getTabState<DeckTabState>(again) as DeckTabState
    expect(state.scheduled.value.path).toBe('')
  })

  it('forgets what it was told about the preset it chose', async () => {
    const one = await open({ notScheduled: 'unreadable' })
    one.tab.setSchedule('Sanskrit.md')
    await settle()
    expect(one.tab.scheduled.value.errorMessage).toBe(words.notScheduled)
    one.tab.close(one.id)
    await settle()

    const again = await one.windowTabs.openTab(DECK, 'Animals.md')
    await settle()

    const state = one.windowTabs.getTabState<DeckTabState>(again) as DeckTabState
    expect(state.scheduled.value.errorMessage).toBe('')
  })

  it('forgets the deck it drew, so the file is read again under new identities', async () => {
    const unmarked = CARDS.map((one) => ({ ...one, cardId: '' }))
    const one = await open({ cards: unmarked })
    const before = one.tab.deck.value.cards.map((card) => card.id)
    one.tab.close(one.id)
    await settle()

    const again = await one.windowTabs.openTab(DECK, 'Animals.md')
    await settle()

    const state = one.windowTabs.getTabState<DeckTabState>(again) as DeckTabState
    const after = state.deck.value.cards.map((card) => card.id)
    expect(after).toHaveLength(2)
    expect(after).not.toStrictEqual(before)
  })

  it('forgets the marks it drew', async () => {
    const one = await open({
      problems: [{ kind: 'unknown', card: 0, template: null, field: '', text: 'Odd' }],
    })
    const before = one.tab.marks.value
    one.tab.close(one.id)
    await settle()

    const again = await one.windowTabs.openTab(DECK, 'Animals.md')
    await settle()

    const state = one.windowTabs.getTabState<DeckTabState>(again) as DeckTabState
    expect(state.marks.value.at.size).toBe(1)
    expect(state.marks.value).not.toBe(before)
  })
})

describe('what the window asks of a deck tab', () => {
  it('is that of the file it remains at', async () => {
    const one = await open()

    expect(one.decks.kind.getOpenTab?.(one.tab)).toStrictEqual({ path: 'Animals.md' })
    expect(one.decks.kind.getTarget?.(one.tab)).toStrictEqual({
      path: 'Animals.md',
      title: 'Animals',
    })
  })

  it('is asking while the file moved past what the deck holds, and not otherwise', async () => {
    const one = await open({ isChanged: true })
    expect(one.decks.kind.isAsking?.(one.tab)).toBe(false)

    one.tab.addCard('Animal', [], null)
    await one.decks.store.flush()

    expect(one.decks.kind.isAsking?.(one.tab)).toBe(true)
  })

  it('writes what is unwritten out when requested to, and answers once it is', async () => {
    const one = await open()
    one.tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)

    await one.decks.kind.flushPending?.(one.tab)

    expect(one.written).toStrictEqual(['Animals.md Llama, Alpaca, Vicuña'])
  })

  it('is written out with the other open files when the window quits', async () => {
    const one = await open()
    one.tab.addCard('Animal', [{ field: 'Name', text: 'Vicuña' }], null)

    await one.buffers.flush()

    expect(one.written).toStrictEqual(['Animals.md Llama, Alpaca, Vicuña'])
  })

  it('makes a deck of the name given, in the folder given', async () => {
    const one = await open()

    expect(await one.decks.kind.createFile?.('zoology', 'Animals')).toStrictEqual({
      ok: true,
      value: { path: 'Animals.md' },
    })
  })
})
