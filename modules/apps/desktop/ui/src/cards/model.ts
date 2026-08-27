/**
 * What a deck and a stencil are while the window holds them, and what each
 * gesture makes of them.
 *
 * The components take names and text and hand them back under identities they
 * were given. Minting those identities, and turning what a person did into the
 * cards and the fields a file is written from, is here, so a test can ask it
 * without a screen.
 */
import { ordered, renamedIn, type CardLanding, type Cut, type Drawn, type Shown } from '@numen/ui'
import type { Carded, Decked, Faced, Offer, Problem, Stencilled, Value } from '../core'

/** An identity a card or a face is drawn under, which no file carries. */
export type Mint = () => string

const minting: Mint = () => crypto.randomUUID()

/** One card as the window holds it: what the file says, under an identity of its own. */
export interface Card extends Carded {
  readonly id: string
}

/** A deck as the window holds it. */
export interface Deck {
  readonly preamble: string
  readonly cards: readonly Card[]
  readonly tail: string
}

/** A stencil as the window holds it: its fields, and its faces under identities. */
export interface Sheet {
  readonly fields: readonly string[]
  readonly faces: readonly Shown[]
}

/** A deck of no cards, which is what a file nothing has been written to holds. */
export const NO_DECK: Deck = { preamble: '', cards: [], tail: '' }

/** A stencil that names nothing and shows nothing. */
export const NO_SHEET: Sheet = { fields: [], faces: [] }

/** A deck as the vault read it, each card under an identity this window mints. */
export const deckOf = (read: Decked, mint: Mint = minting): Deck => ({
  preamble: read.preamble,
  cards: read.cards.map((card) => ({ ...card, id: mint() })),
  tail: read.tail,
})

/** A stencil as the vault read it, each face under an identity this window mints. */
export const sheetOf = (read: Stencilled, mint: Mint = minting): Sheet => ({
  fields: read.fields,
  faces: read.faces.map((face) => ({ id: mint(), ...face })),
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
    cards: deck.cards.map((card) => ({
      id: card.id,
      name: card.name,
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
    faces: sheet.faces.map((face) => ({
      id: face.id,
      name: face.name,
      front: face.front,
      back: face.back,
    })),
  })

/** The deck a string stands for. A string holding nothing is a deck of no cards. */
export const deckIn = (body: string): Deck => (body ? (JSON.parse(body) as Deck) : NO_DECK)

/** The stencil a string stands for, the same way. */
export const sheetIn = (body: string): Sheet => (body ? (JSON.parse(body) as Sheet) : NO_SHEET)

/** The cards of a deck, in the shape the vault takes them. */
export const cardsOf = (deck: Deck): readonly Carded[] =>
  deck.cards.map(({ name, stencil, stencilAt, lead, values }) => ({
    name,
    stencil,
    stencilAt,
    lead,
    values,
  }))

/** The faces of a stencil, in the shape the vault takes them. */
export const facesOf = (sheet: Sheet): readonly Faced[] =>
  sheet.faces.map(({ name, front, back }) => ({ name, front, back }))

/**
 * The cards as the grid draws them, each under the stencil its wikilink
 * resolves to. A card whose link reaches no stencil is drawn under what the
 * file wrote in the brackets.
 */
export const drawnOf = (deck: Deck, offers: readonly Offer[]): readonly Drawn[] => {
  const titles = new Map(offers.map((offer) => [offer.path, offer.title]))
  return deck.cards.map((card) => ({
    id: card.id,
    name: card.name,
    stencil: titles.get(card.stencilAt) ?? card.stencil,
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

/** Where the stencil of that title is filed, and nowhere where none is. */
export const pathOfCut = (offers: readonly Offer[], name: string): string =>
  offers.find((offer) => offer.title === name)?.path ?? ''

/**
 * The field a stencil names its cards by, and nothing for a stencil declaring
 * none. What stands in that field is the card's heading, so it is written and
 * read as the name and never as a value.
 */
export const namingField = (offers: readonly Offer[], stencil: string): string =>
  offers.find((offer) => offer.title === stencil)?.fields[0] ?? ''

/** Whether that field is the one the card is named by. */
export const names = (offers: readonly Offer[], stencil: string, field: string): boolean =>
  field !== '' && namingField(offers, stencil) === field

/**
 * A card added at the end, cut by the stencil filed at a path, with a value
 * standing empty for each field. The wikilink the file carries names that
 * stencil by its title.
 */
export const added = (
  deck: Deck,
  name: string,
  stencil: string,
  stencilAt: string,
  values: readonly Value[],
  mint: Mint = minting,
): Deck => ({
  ...deck,
  cards: [...deck.cards, { id: mint(), name, stencil, stencilAt, lead: '', values }],
})

/** A card taken out of the deck. */
export const removed = (deck: Deck, id: string): Deck => ({
  ...deck,
  cards: deck.cards.filter((card) => card.id !== id),
})

/** A card let go somewhere in the order: before another, or at the end. */
export const carried = (deck: Deck, id: string, at: CardLanding): Deck => {
  const order = ordered(
    deck.cards.map((card) => card.id),
    id,
    at,
  )
  const held = new Map(deck.cards.map((card) => [card.id, card]))
  const cards: Card[] = []
  for (const one of order) {
    const card = held.get(one)
    if (card) cards.push(card)
  }
  return { ...deck, cards }
}

/** A card under another name. */
export const named = (deck: Deck, id: string, name: string): Deck => ({
  ...deck,
  cards: deck.cards.map((card) => (card.id === id ? { ...card, name } : card)),
})

/**
 * One value of one card as it now reads. A field the card has keeps the place
 * it stands in the file; one it does not have is written after the rest, and
 * nothing at all is written for a value with nothing in it.
 */
export const filled = (deck: Deck, id: string, field: string, text: string): Deck => ({
  ...deck,
  cards: deck.cards.map((card) => {
    if (card.id !== id) return card
    if (card.values.some((value) => value.field === field)) {
      return {
        ...card,
        values: card.values.map((value) => (value.field === field ? { field, text } : value)),
      }
    }
    return text === '' ? card : { ...card, values: [...card.values, { field, text }] }
  }),
})

/** A field named at the end of the order. */
export const fieldAdded = (sheet: Sheet, name: string): Sheet => ({
  ...sheet,
  fields: [...sheet.fields, name],
})

/**
 * A field under another name. The name is written twice over — once here and
 * once in every face that stands it — so the faces are rewritten with it.
 */
export const fieldNamed = (sheet: Sheet, field: string, name: string): Sheet => ({
  fields: sheet.fields.map((one) => (one === field ? name : one)),
  faces: sheet.faces.map((face) => ({
    ...face,
    front: renamedIn(face.front, field, name),
    back: renamedIn(face.back, field, name),
  })),
})

/** A field the stencil no longer names. What the faces stand in its braces stays. */
export const fieldGone = (sheet: Sheet, field: string): Sheet => ({
  ...sheet,
  fields: sheet.fields.filter((one) => one !== field),
})

/** A field let go somewhere in the order. */
export const fieldCarried = (sheet: Sheet, field: string, at: CardLanding): Sheet => ({
  ...sheet,
  fields: ordered(sheet.fields, field, at),
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
  faces: [...sheet.faces, { id: mint(), name, front: '', back: '' }],
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
  /** What is wrong with each field, under the name the file spells it. */
  readonly fields: ReadonlyMap<string, readonly string[]>
  /** What is wrong that stands against no card, no face and no field. */
  readonly whole: readonly string[]
}

/**
 * Each problem against the thing it stands on. A problem carries where it
 * stands rather than what it is called, so a card with no name and two cards of
 * one name each land on the tile they were read from. One standing past the end
 * of what was read is against the file.
 */
export const marksOf = (
  problems: readonly Problem[],
  cards: readonly string[],
  faces: readonly string[],
): Marks => {
  const at = new Map<string, string[]>()
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
      against(at, card, problem.text)
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

  return { at, fields, whole }
}
