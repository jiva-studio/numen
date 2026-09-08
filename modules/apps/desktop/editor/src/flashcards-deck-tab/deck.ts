/**
 * What a deck is while the window holds it, and what each gesture makes of it.
 *
 * The components take names and text and hand them back under identities they
 * were given. Minting those identities, and turning what a person did into the
 * cards and the sections a file is written from, is here.
 */
import {
  CARD_HEAD,
  cardEnded,
  type DeckSection,
  type InsertionPoint,
  type Stencil,
  type DeckCard,
} from '@numen/ui'
import type {
  VaultCard,
  VaultDeck,
  VaultSection,
  StencilSummary,
  Value,
} from '../shared/flashcards/cards'
import { minting, type IdMaker } from '../shared/flashcards/identity'
import { nameOf } from '../shared/paths'

/**
 * One card as the window holds it: what the file says, under the identity it is
 * addressed by.
 */
export interface BufferCard {
  /** The identity the window addresses it by: its mark, or one minted for it. */
  readonly id: string
  /**
   * What the card is, for as long as it exists, without the caret its heading
   * writes it behind. Empty for a card the application has not written yet.
   */
  readonly mark: string
  /**
   * The section it stands under, by the identity this window knows that section
   * at. Nothing for a card standing before the first.
   */
  readonly section: string | null
  /**
   * The line its heading says, with the mark taken off. Nothing draws it and
   * nothing is typed into it: a write reads it again from the first field.
   */
  readonly heading: string
  /**
   * The stencil it is cut by, as the wikilink beneath its heading names it: a
   * name where one picks the stencil, and `note://<identifier>` where none does.
   */
  readonly stencilLink: string
  /**
   * Where that stencil is filed, as the wikilink resolved in the vault. Empty
   * for a card naming none and for a name that reaches no note.
   */
  readonly stencilPath: string
  /** The prose between that wikilink and the first field. */
  readonly preamble: string
  readonly values: readonly Value[]
}

/** One section as the window holds it, under an identity of its own. */
export interface BufferSection {
  /** The identity the window addresses it by, minted at every reading. */
  readonly id: string
  /** What it is called, as its heading spells it. Two sections may carry one name. */
  readonly name: string
  /** The prose between its heading and its first card. */
  readonly preamble: string
}

/** A deck as the window holds it. */
export interface BufferDeck {
  /** The prose below the frontmatter and above the first card or section. */
  readonly preamble: string
  readonly cards: readonly BufferCard[]
  readonly sections: readonly BufferSection[]
  /** What the file ends with once the last card has been read. */
  readonly tail: string
}

/** A deck of no cards, which is what a file nothing has been written to holds. */
export const NO_DECK: BufferDeck = { preamble: '', cards: [], sections: [], tail: '' }

/**
 * A deck as the vault read it.
 *
 * A card is known by its mark, which the file carries and which follows the
 * card wherever it goes. A card the file cannot name — one carrying no mark,
 * and each of two carrying one mark between them — is known by an identity this
 * window mints for the reading, so that both of two are drawn and each is typed
 * into on its own. A section carries no mark, so every one of them is minted.
 *
 * A card standing under a section this reading does not hold stands before the
 * first section, where it is drawn and where the next write puts it.
 */
export const deckOf = (read: VaultDeck, mint: IdMaker = minting): BufferDeck => {
  const held = new Map<string, number>()
  for (const card of read.cards) held.set(card.mark, (held.get(card.mark) ?? 0) + 1)
  const sections = read.sections.map((section) => ({
    id: mint(),
    name: section.name,
    preamble: section.preamble,
  }))
  return {
    preamble: read.preamble,
    cards: read.cards.map((card) => ({
      id: card.mark && held.get(card.mark) === 1 ? card.mark : mint(),
      mark: card.mark,
      // The window addresses a section by the identity it minted for it, and
      // the place the file gave it goes no further than here.
      section: card.sectionIndex === null ? null : (sections[card.sectionIndex]?.id ?? null),
      heading: card.heading,
      stencilLink: card.stencilLink,
      stencilPath: card.stencilPath,
      preamble: card.preamble,
      values: card.values,
    })),
    sections,
    tail: read.tail,
  }
}

/**
 * A deck as one string, which is what the tab holding it is dirty against. The
 * parts are written in a settled order, so a deck that came back unchanged
 * reads as the string it went in as.
 */
export const deckBodyOf = (deck: BufferDeck): string =>
  JSON.stringify({
    preamble: deck.preamble,
    tail: deck.tail,
    sections: deck.sections.map((section) => ({
      id: section.id,
      name: section.name,
      preamble: section.preamble,
    })),
    cards: deck.cards.map((card) => ({
      id: card.id,
      mark: card.mark,
      section: card.section,
      heading: card.heading,
      stencilLink: card.stencilLink,
      stencilPath: card.stencilPath,
      preamble: card.preamble,
      values: card.values.map((value) => ({ field: value.field, text: value.text })),
    })),
  })

/** The deck a string stands for. A string holding nothing is a deck of no cards. */
export const deckIn = (body: string): BufferDeck => (body ? (JSON.parse(body) as BufferDeck) : NO_DECK)

/**
 * The cards of a deck, in the shape the vault takes them. A card says where the
 * section it stands under stands in the deck's own, which is what the file
 * writes it under.
 */
export const cardsOf = (deck: BufferDeck): readonly VaultCard[] => {
  const at = new Map(deck.sections.map((section, index) => [section.id, index]))
  return deck.cards.map(({ mark, section, heading, stencilLink, stencilPath, preamble, values }) => ({
    mark,
    sectionIndex: section === null ? null : (at.get(section) ?? null),
    heading,
    stencilLink,
    stencilPath,
    preamble,
    values,
  }))
}

/** The sections of a deck, in the shape the vault takes them. */
export const sectionsOf = (deck: BufferDeck): readonly VaultSection[] =>
  deck.sections.map(({ name, preamble }) => ({ name, preamble }))

/** The sections as the grid draws them, each under the identity it was read at. */
export const drawnSectionsOf = (deck: BufferDeck): readonly DeckSection[] =>
  deck.sections.map(({ id, name }) => ({ id, name }))

/**
 * The cards as the grid draws them, each under the stencil its wikilink
 * resolves to. A card whose link reaches no stencil is drawn under what the
 * file wrote in the brackets, and a card that wrote nothing there is cut by
 * nothing.
 */
export const drawnOf = (deck: BufferDeck, offers: readonly StencilSummary[]): readonly DeckCard[] => {
  const titles = new Map(offers.map((offer) => [offer.path, offer.title]))
  return deck.cards.map((card) => ({
    id: card.id,
    section: card.section,
    stencil: (titles.get(card.stencilPath) ?? card.stencilLink) || null,
    filled: card.values.map((value) => ({ field: value.field, text: value.text })),
  }))
}

/**
 * The stencils a card may be cut by, under the word a card names one by. Two
 * stencils of one title name one stencil, and the first stands.
 */
export const stencilsOf = (offers: readonly StencilSummary[]): readonly Stencil[] => {
  const taken = new Set<string>()
  const stencils: Stencil[] = []
  for (const offer of offers) {
    if (!offer.title || taken.has(offer.title)) continue
    taken.add(offer.title)
    stencils.push({ name: offer.title, fields: offer.fields })
  }
  return stencils
}

/**
 * Whether two decks read the same. What a person can see is the prose and the
 * cards; the identities this window mints are its own, and a file read again
 * carries fresh ones for the same cards.
 *
 * A heading is read back from its card's first field wherever the deck is
 * written, so a save carrying a value somebody typed comes back under a heading
 * this window never held. Two readings that differ in that alone are one
 * reading, and the card being typed into stands.
 */
export const sameDeck = (one: BufferDeck, other: BufferDeck): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(sectionsOf(one)) === JSON.stringify(sectionsOf(other)) &&
  JSON.stringify(written(one)) === JSON.stringify(written(other))

/** The cards as somebody wrote them, without the heading a write reads back. */
const written = (deck: BufferDeck): readonly Omit<VaultCard, 'heading'>[] =>
  cardsOf(deck).map(({ heading: _heading, ...card }) => card)

/**
 * The deck on screen under the headings the file now carries. Nothing draws a
 * heading and nothing is typed into one, so a deck that reads the same but for
 * its headings takes them and stands: what the next write puts in the file is
 * what the file says, and not what it said when the deck was drawn.
 */
export const headed = (held: BufferDeck, read: BufferDeck): BufferDeck => {
  const carried = (at: number): string => read.cards[at]?.heading ?? ''
  if (held.cards.every((card, at) => card.heading === carried(at))) return held
  return { ...held, cards: held.cards.map((card, at) => ({ ...card, heading: carried(at) })) }
}

/**
 * A reading of a deck under the identities the window already drew it by. A
 * card the file could not name was drawn under an identity this window minted,
 * and the reading that names it is that same card: it keeps the identity it is
 * being typed into under, and the tile it stands in is not drawn again.
 */
export const named = (held: BufferDeck, read: BufferDeck): BufferDeck => {
  /** Where a card's section stands among its deck's, each reading minting its own. */
  const seat = (deck: BufferDeck, section: string | null): number =>
    section === null ? -1 : deck.sections.findIndex((each) => each.id === section)

  // A section carries no mark, so every reading mints one for it. The grid
  // draws a run under the identity its section stands at, so a section still
  // standing where it stood keeps the identity it is drawn under.
  const sections =
    held.sections.length === read.sections.length
      ? read.sections.map((section, at) => {
          const was = held.sections[at]
          return was && was.name === section.name && was.preamble === section.preamble
            ? { ...section, id: was.id }
            : section
        })
      : read.sections

  /** The section a card of the reading now stands under, by the identity kept. */
  const under = (section: string | null): string | null =>
    section === null ? null : (sections[seat(read, section)]?.id ?? null)

  // A card is the card it was where the reading holds as many as the window
  // does; a reading of a different length names no card the window drew.
  const alongside = held.cards.length === read.cards.length

  const cards = read.cards.map((card, at) => {
    const stands = { ...card, section: under(card.section) }
    const was = held.cards[at]
    if (!alongside || !was || was.mark !== '' || card.mark === '') return stands
    const same =
      was.stencilLink === card.stencilLink &&
      was.preamble === card.preamble &&
      seat(held, was.section) === seat(read, card.section) &&
      JSON.stringify(was.values) === JSON.stringify(card.values)
    return same ? { ...stands, id: was.id } : stands
  })

  return { ...read, sections, cards }
}

/**
 * Whether two listings name the same stencils, in the same order and with the
 * same fields.
 */
export const sameOffers = (one: readonly StencilSummary[], other: readonly StencilSummary[]): boolean =>
  one.length === other.length &&
  one.every((offer, at) => {
    const against = other[at]
    return (
      against !== undefined &&
      offer.path === against.path &&
      offer.title === against.title &&
      offer.fields.length === against.fields.length &&
      offer.fields.every((field, seat) => field === against.fields[seat])
    )
  })

/** Where the stencil of that title is filed, and nowhere where none is. */
export const pathOfCut = (offers: readonly StencilSummary[], name: string): string =>
  offers.find((offer) => offer.title === name)?.path ?? ''


/**
 * A card made at the end of a section, cut by the stencil filed at a path, with
 * a value standing empty for each field. Under no section it is made at the end
 * of the cards standing before the first of them. The wikilink the file carries
 * names that stencil by its file's name, and by its title where the vault filed
 * it nowhere.
 *
 * It carries no mark: a mark is written where the deck is made whole, on its
 * way to the vault, and the window minted an identity to draw it under until
 * the file names it.
 */
export const added = (
  deck: BufferDeck,
  title: string,
  stencilPath: string,
  values: readonly Value[],
  section: string | null = null,
  mint: IdMaker = minting,
): BufferDeck => {
  // A card is filed where its section stands, so it goes after every card of
  // that section and of the ones before it.
  const ranks = new Map(deck.sections.map((each, index) => [each.id, index]))
  const rank = (id: string | null): number => (id === null ? -1 : (ranks.get(id) ?? -1))
  const mine = rank(section)

  let at = 0
  deck.cards.forEach((card, index) => {
    if (rank(card.section) <= mine) at = index + 1
  })

  return {
    ...deck,
    cards: [
      ...deck.cards.slice(0, at),
      {
        id: mint(),
        mark: '',
        section: section !== null && ranks.has(section) ? section : null,
        heading: '',
        stencilLink: stencilPath ? nameOf(stencilPath) : title,
        stencilPath,
        preamble: '',
        values,
      },
      ...deck.cards.slice(at),
    ],
  }
}

/** A card taken out of the deck. */
export const removed = (deck: BufferDeck, id: string): BufferDeck => ({
  ...deck,
  cards: deck.cards.filter((card) => card.id !== id),
})

/**
 * A card let go somewhere in the deck: before the card of that identity, at the
 * head of the deck, at the head of the section of that identity, past the last
 * card standing under a heading, or at the end. A card takes the section of
 * whatever it lands in front of, one let go at the head of the deck stands
 * under no section, one let go past the last card under a heading stands under
 * that heading, and one let go at the end stands under the last section.
 */
export const dropped = (deck: BufferDeck, id: string, at: InsertionPoint): BufferDeck => {
  const held = deck.cards.find((card) => card.id === id)
  if (!held) return deck
  const left = deck.cards.filter((card) => card.id !== id)

  if (at === CARD_HEAD) return { ...deck, cards: [{ ...held, section: null }, ...left] }

  const before = at === null ? -1 : left.findIndex((card) => card.id === at)
  if (before !== -1) {
    const under = { ...held, section: left[before]?.section ?? null }
    return { ...deck, cards: [...left.slice(0, before), under, ...left.slice(before)] }
  }

  if (at === null) {
    return { ...deck, cards: [...left, { ...held, section: deck.sections.at(-1)?.id ?? null }] }
  }

  // The cards stand grouped in the order of the sections, so the head of one is
  // where the first card standing that far down the deck stands, and the end of
  // one is past the last card standing under it.
  const rank = (section: string | null): number =>
    section === null ? 0 : deck.sections.findIndex((each) => each.id === section) + 1
  const head = (section: string | null): number => {
    const seat = left.findIndex((card) => rank(card.section) >= rank(section))
    return seat === -1 ? left.length : seat
  }
  const put = (section: string | null, where: number): BufferDeck => ({
    ...deck,
    cards: [...left.slice(0, where), { ...held, section }, ...left.slice(where)],
  })

  // A card naming a section the deck does not hold stands before the first
  // heading, which is where it is counted from.
  const under = (card: BufferCard): string | null =>
    deck.sections.some((each) => each.id === card.section) ? card.section : null

  const last = (section: string | null): number => {
    let seat = -1
    left.forEach((card, index) => {
      if (under(card) === section) seat = index
    })
    return seat
  }

  const run = cardEnded(at)
  if (run !== null) {
    const section = run === CARD_HEAD ? null : run
    if (section !== null && !deck.sections.some((each) => each.id === section)) return deck
    const seat = last(section)
    return put(section, seat === -1 ? head(section) : seat + 1)
  }

  if (!deck.sections.some((section) => section.id === at)) return deck
  return put(at, head(at))
}

/** A section made at the end of the deck, holding no card. */
export const sectionAdded = (deck: BufferDeck, name: string, mint: IdMaker = minting): BufferDeck => ({
  ...deck,
  sections: [...deck.sections, { id: mint(), name, preamble: '' }],
})

/** A section under another name. */
export const sectionNamed = (deck: BufferDeck, id: string, name: string): BufferDeck => ({
  ...deck,
  sections: deck.sections.map((section) =>
    section.id === id ? { ...section, name } : section,
  ),
})

/** One piece of a deck's prose after another, with a line between the two. */
const after = (above: string, below: string): string =>
  above && below ? `${above.trimEnd()}\n\n${below}` : above || below

/**
 * A section taken out of the deck. A section is a name, so taking it away takes
 * away its heading and nothing else: its cards stay where they stand, under
 * whatever heading is above them now, and the text it stood on stays too, after
 * the text of the section above it or after the deck's own where none stands
 * above.
 */
export const sectionGone = (deck: BufferDeck, id: string): BufferDeck => {
  const at = deck.sections.findIndex((section) => section.id === id)
  if (at === -1) return deck
  const going = deck.sections[at]
  const above = deck.sections[at - 1]
  return {
    ...deck,
    preamble: above ? deck.preamble : after(deck.preamble, going?.preamble ?? ''),
    sections: deck.sections.flatMap((section) => {
      if (section.id === id) return []
      if (above && section.id === above.id) {
        return [{ ...section, preamble: after(section.preamble, going?.preamble ?? '') }]
      }
      return [section]
    }),
    cards: deck.cards.map((card) =>
      card.section === id ? { ...card, section: above?.id ?? null } : card,
    ),
  }
}

/**
 * One value of one card as it now reads. A field the card has keeps the place
 * it stands in the file; one it does not have is written after the rest, and
 * nothing at all is written for a value with nothing in it.
 */
export const filled = (
  deck: BufferDeck,
  id: string,
  field: string,
  nth: number,
  text: string,
): BufferDeck => ({
  ...deck,
  cards: deck.cards.map((card) => {
    if (card.id !== id) return card

    // A card writing one field twice holds two values, and the one typed in is
    // the one counted off under that field.
    let under = 0
    let found = false
    const values = card.values.map((value) => {
      if (value.field !== field) return value
      under += 1
      if (under !== nth) return value
      found = true
      return { field, text }
    })
    if (found) return { ...card, values }
    return text === '' ? card : { ...card, values: [...card.values, { field, text }] }
  }),
})
