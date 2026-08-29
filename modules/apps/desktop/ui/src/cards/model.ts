/**
 * What a deck and a stencil are while the window holds them, and what each
 * gesture makes of them.
 *
 * The components take names and text and hand them back under identities they
 * were given. Minting those identities, and turning what a person did into the
 * cards and the fields a file is written from, is here, so a test can ask it
 * without a screen.
 */
import {
  CARD_HEAD,
  cardEnded,
  ordered,
  reordered,
  type Banded,
  type CardLanding,
  type Cut,
  type Drawn,
} from '@numen/ui'
import type {
  Carded,
  Decked,
  Faced,
  Offer,
  Problem,
  Sectioned,
  Stencilled,
  Value,
} from '../core'

/** An identity something is drawn under, which no file carries. */
export type Mint = () => string

const minting: Mint = () => crypto.randomUUID()

/**
 * One card as the window holds it: what the file says, under the identity it is
 * addressed by.
 */
export interface Card extends Omit<Carded, 'section'> {
  readonly id: string
  /**
   * The section it stands under, by the identity this window knows that section
   * at. Nothing for a card standing before the first.
   */
  readonly section: string | null
}

/** One section as the window holds it, under an identity of its own. */
export interface Section extends Sectioned {
  readonly id: string
}

/** A deck as the window holds it. */
export interface Deck {
  readonly preamble: string
  readonly cards: readonly Card[]
  readonly sections: readonly Section[]
  readonly tail: string
}

/** One face as the window holds it: what the file says, under an identity of its own. */
export interface Face extends Faced {
  readonly id: string
}

/** A stencil as the window holds it: its fields, and its faces under identities. */
export interface Sheet {
  readonly fields: readonly string[]
  readonly preamble: string
  readonly faces: readonly Face[]
  readonly tail: string
}

/** A deck of no cards, which is what a file nothing has been written to holds. */
export const NO_DECK: Deck = { preamble: '', cards: [], sections: [], tail: '' }

/** A stencil that names nothing and shows nothing. */
export const NO_SHEET: Sheet = { fields: [], preamble: '', faces: [], tail: '' }

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
export const deckOf = (read: Decked, mint: Mint = minting): Deck => {
  const held = new Map<string, number>()
  for (const card of read.cards) held.set(card.mark, (held.get(card.mark) ?? 0) + 1)
  const sections = read.sections.map((section) => ({ ...section, id: mint() }))
  return {
    preamble: read.preamble,
    cards: read.cards.map((card) => ({
      ...card,
      id: card.mark && held.get(card.mark) === 1 ? card.mark : mint(),
      section: card.section === null ? null : (sections[card.section]?.id ?? null),
    })),
    sections,
    tail: read.tail,
  }
}

/** A stencil as the vault read it, each face under an identity this window mints. */
export const sheetOf = (read: Stencilled, mint: Mint = minting): Sheet => ({
  fields: read.fields,
  preamble: read.preamble,
  faces: read.faces.map((face) => ({ id: mint(), ...face })),
  tail: read.tail,
})

/**
 * A deck as one string, which is what the tab holding it is dirty against. The
 * parts are written in a settled order, so a deck that came back unchanged
 * reads as the string it went in as.
 */
export const bodyOf = (deck: Deck): string =>
  JSON.stringify({
    preamble: deck.preamble,
    tail: deck.tail,
    sections: deck.sections.map((section) => ({
      id: section.id,
      name: section.name,
      lead: section.lead,
    })),
    cards: deck.cards.map((card) => ({
      id: card.id,
      mark: card.mark,
      section: card.section,
      heading: card.heading,
      stencil: card.stencil,
      stencilAt: card.stencilAt,
      lead: card.lead,
      values: card.values.map((value) => ({ field: value.field, text: value.text })),
    })),
  })

/** A stencil as one string, the same way. */
export const sheetBodyOf = (sheet: Sheet): string =>
  JSON.stringify({
    fields: sheet.fields,
    preamble: sheet.preamble,
    tail: sheet.tail,
    faces: sheet.faces.map((face) => ({
      id: face.id,
      name: face.name,
      lead: face.lead,
      front: face.front,
      back: face.back,
    })),
  })

/** The deck a string stands for. A string holding nothing is a deck of no cards. */
export const deckIn = (body: string): Deck => (body ? (JSON.parse(body) as Deck) : NO_DECK)

/** The stencil a string stands for, the same way. */
export const sheetIn = (body: string): Sheet => (body ? (JSON.parse(body) as Sheet) : NO_SHEET)

/**
 * The cards of a deck, in the shape the vault takes them. A card says where the
 * section it stands under stands in the deck's own, which is what the file
 * writes it under.
 */
export const cardsOf = (deck: Deck): readonly Carded[] => {
  const at = new Map(deck.sections.map((section, index) => [section.id, index]))
  return deck.cards.map(({ mark, section, heading, stencil, stencilAt, lead, values }) => ({
    mark,
    section: section === null ? null : (at.get(section) ?? null),
    heading,
    stencil,
    stencilAt,
    lead,
    values,
  }))
}

/** The sections of a deck, in the shape the vault takes them. */
export const sectionsOf = (deck: Deck): readonly Sectioned[] =>
  deck.sections.map(({ name, lead }) => ({ name, lead }))

/** The sections as the grid draws them, each under the identity it was read at. */
export const bandedOf = (deck: Deck): readonly Banded[] =>
  deck.sections.map(({ id, name }) => ({ id, name }))

/** The faces of a stencil, in the shape the vault takes them. */
export const facesOf = (sheet: Sheet): readonly Faced[] =>
  sheet.faces.map(({ name, lead, front, back }) => ({ name, lead, front, back }))

/**
 * The cards as the grid draws them, each under the stencil its wikilink
 * resolves to. A card whose link reaches no stencil is drawn under what the
 * file wrote in the brackets, and a card that wrote nothing there is cut by
 * nothing.
 */
export const drawnOf = (deck: Deck, offers: readonly Offer[]): readonly Drawn[] => {
  const titles = new Map(offers.map((offer) => [offer.path, offer.title]))
  return deck.cards.map((card) => ({
    id: card.id,
    section: card.section,
    stencil: (titles.get(card.stencilAt) ?? card.stencil) || null,
    filled: card.values.map((value) => ({ field: value.field, text: value.text })),
  }))
}

/**
 * The stencils a card may be cut by, under the word a card names one by. Two
 * stencils of one title name one stencil, and the first stands.
 */
export const cutsOf = (offers: readonly Offer[]): readonly Cut[] => {
  const taken = new Set<string>()
  const cuts: Cut[] = []
  for (const offer of offers) {
    if (!offer.title || taken.has(offer.title)) continue
    taken.add(offer.title)
    cuts.push({ name: offer.title, fields: offer.fields })
  }
  return cuts
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
export const sameDeck = (one: Deck, other: Deck): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(sectionsOf(one)) === JSON.stringify(sectionsOf(other)) &&
  JSON.stringify(written(one)) === JSON.stringify(written(other))

/** The cards as somebody wrote them, without the heading a write reads back. */
const written = (deck: Deck): readonly Omit<Carded, 'heading'>[] =>
  cardsOf(deck).map(({ heading: _heading, ...card }) => card)

/**
 * The deck on screen under the headings the file now carries. Nothing draws a
 * heading and nothing is typed into one, so a deck that reads the same but for
 * its headings takes them and stands: what the next write puts in the file is
 * what the file says, and not what it said when the deck was drawn.
 */
export const headed = (held: Deck, read: Deck): Deck => {
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
export const named = (held: Deck, read: Deck): Deck => {
  /** Where a card's section stands among its deck's, each reading minting its own. */
  const seat = (deck: Deck, section: string | null): number =>
    section === null ? -1 : deck.sections.findIndex((each) => each.id === section)

  // A section carries no mark, so every reading mints one for it. The grid
  // draws a run under the identity its section stands at, so a section still
  // standing where it stood keeps the identity it is drawn under.
  const sections =
    held.sections.length === read.sections.length
      ? read.sections.map((section, at) => {
          const was = held.sections[at]
          return was && was.name === section.name && was.lead === section.lead
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
      was.stencil === card.stencil &&
      was.stencilAt === card.stencilAt &&
      was.lead === card.lead &&
      seat(held, was.section) === seat(read, card.section) &&
      JSON.stringify(was.values) === JSON.stringify(card.values)
    return same ? { ...stands, id: was.id } : stands
  })

  return { ...read, sections, cards }
}

/** Whether two stencils read the same, the identities left out the same way. */
export const sameSheet = (one: Sheet, other: Sheet): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(one.fields) === JSON.stringify(other.fields) &&
  JSON.stringify(facesOf(one)) === JSON.stringify(facesOf(other))

/** Whether two lists of words read the same, in the same order. */
const sameWords = (one: readonly string[], other: readonly string[]): boolean =>
  one.length === other.length && one.every((text, at) => text === other[at])

/** Whether two of those maps stand against the same things, saying the same. */
const sameAgainst = (
  one: ReadonlyMap<string, readonly string[]>,
  other: ReadonlyMap<string, readonly string[]>,
): boolean => {
  if (one.size !== other.size) return false
  for (const [key, said] of one) {
    const against = other.get(key)
    if (!against || !sameWords(said, against)) return false
  }
  return true
}

/**
 * Whether two readings of a file are wrong in the same way. What is drawn
 * against a file it says nothing about is drawn again.
 */
export const sameMarks = (one: Marks, other: Marks): boolean =>
  sameWords(one.whole, other.whole) &&
  sameAgainst(one.at, other.at) &&
  sameUnder(one.under, other.under) &&
  sameAgainst(one.fields, other.fields)

/** Whether two of those hold the same fields of the same cards, saying the same. */
const sameUnder = (
  one: ReadonlyMap<string, ReadonlyMap<string, readonly string[]>>,
  other: ReadonlyMap<string, ReadonlyMap<string, readonly string[]>>,
): boolean => {
  if (one.size !== other.size) return false
  for (const [key, said] of one) {
    const against = other.get(key)
    if (!against || !sameAgainst(said, against)) return false
  }
  return true
}

/**
 * Whether two listings name the same stencils, in the same order and with the
 * same fields.
 */
export const sameOffers = (one: readonly Offer[], other: readonly Offer[]): boolean =>
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
export const pathOfCut = (offers: readonly Offer[], name: string): string =>
  offers.find((offer) => offer.title === name)?.path ?? ''

/**
 * How a card names the stencil filed at a path: the file's name, without the
 * folders above it and without the extension. A name is what a link resolves
 * by, so the link stands where the stencil is moved.
 */
export const linkTo = (path: string): string => {
  const file = path.split('/').pop() ?? ''
  const dot = file.lastIndexOf('.')
  return dot > 0 ? file.slice(0, dot) : file
}

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
  deck: Deck,
  title: string,
  stencilAt: string,
  values: readonly Value[],
  section: string | null = null,
  mint: Mint = minting,
): Deck => {
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
        stencil: stencilAt ? linkTo(stencilAt) : title,
        stencilAt,
        lead: '',
        values,
      },
      ...deck.cards.slice(at),
    ],
  }
}

/** A card taken out of the deck. */
export const removed = (deck: Deck, id: string): Deck => ({
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
export const carried = (deck: Deck, id: string, at: CardLanding): Deck => {
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
  const put = (section: string | null, where: number): Deck => ({
    ...deck,
    cards: [...left.slice(0, where), { ...held, section }, ...left.slice(where)],
  })

  // A card naming a section the deck does not hold stands before the first
  // heading, which is where it is counted from.
  const under = (card: Card): string | null =>
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
export const sectionAdded = (deck: Deck, name: string, mint: Mint = minting): Deck => ({
  ...deck,
  sections: [...deck.sections, { id: mint(), name, lead: '' }],
})

/** A section under another name. */
export const sectionNamed = (deck: Deck, id: string, name: string): Deck => ({
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
export const sectionGone = (deck: Deck, id: string): Deck => {
  const at = deck.sections.findIndex((section) => section.id === id)
  if (at === -1) return deck
  const going = deck.sections[at]
  const above = deck.sections[at - 1]
  return {
    ...deck,
    preamble: above ? deck.preamble : after(deck.preamble, going?.lead ?? ''),
    sections: deck.sections.flatMap((section) => {
      if (section.id === id) return []
      if (above && section.id === above.id) {
        return [{ ...section, lead: after(section.lead, going?.lead ?? '') }]
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
  deck: Deck,
  id: string,
  field: string,
  nth: number,
  text: string,
): Deck => ({
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

/** A field named at the end of the order. */
export const fieldAdded = (sheet: Sheet, name: string): Sheet => ({
  ...sheet,
  fields: [...sheet.fields, name],
})

/** A field the stencil no longer names. What the faces stand in its braces stays. */
export const fieldGone = (sheet: Sheet, field: string): Sheet => ({
  ...sheet,
  fields: sheet.fields.filter((one) => one !== field),
})

/**
 * A field let go somewhere in the order. The first field names every card the
 * stencil cuts, so it stays first and nothing lands above it.
 */
export const fieldCarried = (sheet: Sheet, field: string, at: CardLanding): Sheet => ({
  ...sheet,
  fields: reordered(sheet.fields, field, at),
})

/**
 * A face let go somewhere in the order: before another, or at the end. The
 * order of the faces is the order a card's repetitions are taken from it, and
 * nothing among them is fixed.
 */
export const faceCarried = (sheet: Sheet, id: string, at: CardLanding): Sheet => {
  const order = ordered(
    sheet.faces.map((face) => face.id),
    id,
    at,
  )
  const held = new Map(sheet.faces.map((face) => [face.id, face]))
  return { ...sheet, faces: order.flatMap((one) => held.get(one) ?? []) }
}

/** A face added at the end, with both its halves empty. */
export const faceAdded = (sheet: Sheet, name: string, mint: Mint = minting): Sheet => ({
  ...sheet,
  faces: [...sheet.faces, { id: mint(), name, lead: '', front: '', back: '' }],
})

/** A face under another name. */
export const faceNamed = (sheet: Sheet, id: string, name: string): Sheet => ({
  ...sheet,
  faces: sheet.faces.map((face) => (face.id === id ? { ...face, name } : face)),
})

/** A face taken out of the stencil. */
export const faceGone = (sheet: Sheet, id: string): Sheet => ({
  ...sheet,
  faces: sheet.faces.filter((face) => face.id !== id),
})

/** One half of one face as it now reads. */
export const faceWritten = (
  sheet: Sheet,
  id: string,
  half: 'front' | 'back',
  text: string,
): Sheet => ({
  ...sheet,
  faces: sheet.faces.map((face) => (face.id === id ? { ...face, [half]: text } : face)),
})

/**
 * Where a mark is drawn: the one element carrying that value in that
 * attribute. A field is named by whatever a person typed, so the quote and the
 * backslash are written out.
 */
export const standingIn = (attribute: string, value: string): string =>
  `[${attribute}="${value.replace(/[\\"]/gu, (char) => `\\${char}`)}"]`

/** Where each problem is drawn. */
export interface Marks {
  /**
   * What is wrong with each card and each face, under the identity this window
   * gave it.
   */
  readonly at: ReadonlyMap<string, readonly string[]>
  /**
   * What is wrong with one field of one card, under that card's identity and
   * then the name the file spells the field.
   */
  readonly under: ReadonlyMap<string, ReadonlyMap<string, readonly string[]>>
  /** What is wrong with each field, under the name the file spells it. */
  readonly fields: ReadonlyMap<string, readonly string[]>
  /** What is wrong that stands against no card, no face and no field. */
  readonly whole: readonly string[]
}

/**
 * Each problem against the thing it stands on. A problem carries where it
 * stands, so a card under no stencil and two cards of one mark each land on the
 * tile they were read from, and a problem naming a field lands on that field of
 * that card. One standing past the end of what was read is against the file.
 */
export const marksOf = (
  problems: readonly Problem[],
  cards: readonly string[],
  faces: readonly string[],
): Marks => {
  const at = new Map<string, string[]>()
  const under = new Map<string, Map<string, string[]>>()
  const fields = new Map<string, string[]>()
  const whole: string[] = []

  const against = (held: Map<string, string[]>, key: string, text: string): void => {
    const said = held.get(key)
    if (said) said.push(text)
    else held.set(key, [text])
  }

  for (const problem of problems) {
    const card = problem.card === null ? undefined : cards[problem.card]
    if (card !== undefined) {
      if (problem.field === '') {
        against(at, card, problem.text)
        continue
      }
      let held = under.get(card)
      if (!held) {
        held = new Map<string, string[]>()
        under.set(card, held)
      }
      against(held, problem.field, problem.text)
      continue
    }
    const face = problem.face === null ? undefined : faces[problem.face]
    if (face !== undefined) {
      against(at, face, problem.text)
      continue
    }
    if (problem.field !== '') {
      against(fields, problem.field, problem.text)
      continue
    }
    whole.push(problem.text)
  }

  return { at, under, fields, whole }
}
