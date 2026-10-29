/**
 * The deck one tab displays, read from the string the store holds, and the
 * problems of the file against the cards the grid displays.
 *
 * Both are compared against what they were read from. A deck that reads equal to
 * the displayed one keeps the displayed one, so the card being edited is not
 * rendered again under a new identity.
 */
import { type DeckProblem, createMarks, areMarksEqual, type Marks } from '../../cards'
import {
  deserializeBufferDeckFromString,
  applyReadHeadings,
  applyScreenIds,
  areDecksEqual,
  type BufferDeck,
} from '../lib/deck'

export interface ShownStore {
  getOpenNote(id: string): { body: string }
  getPath(id: string): string
}

/** The reader one window has, over the decks that window holds. */
export function createDeckReader(
  store: ShownStore,
  getProblems: (path: string) => readonly DeckProblem[],
) {
  /** The last string a deck was read out of, and what it came to. */
  const parsed = new Map<string, { body: string; deck: BufferDeck }>()

  /** The deck one tab is showing, read out of the string the store holds. */
  const getDeck = (id: string): BufferDeck => {
    const body = store.getOpenNote(id).body
    const target = parsed.get(id)
    if (target && target.body === body) return target.deck
    // A file read again carries fresh identities for the same cards, so the
    // string it comes back as differs from the string that went out. A deck
    // reading as the one on screen leaves that one standing, and a deck the
    // file has named a card of since keeps that card's identity, so the card a
    // person is typing into is not drawn again.
    const readItem = deserializeBufferDeckFromString(body)
    let deck = readItem
    if (target) {
      deck = areDecksEqual(target.deck, readItem)
        ? applyReadHeadings(target.deck, readItem)
        : applyScreenIds(target.deck, readItem)
    }
    parsed.set(id, { body, deck })
    return deck
  }

  /** The problems one tab was last marked from, and the marks that came of it. */
  const marked = new Map<string, { problems: readonly DeckProblem[]; marks: Marks }>()

  /**
   * What is wrong with a file, against the card the grid displays. A problem
   * carries its position in the file it was read from, so it is applied to a
   * card once, when the reading it arrived in is the newest: a card moved
   * elsewhere in the order or a card removed next to it takes its mark
   * with it.
   */
  const getMarks = (id: string): Marks => {
    const problems = getProblems(store.getPath(id))
    const target = marked.get(id)
    if (target && target.problems === problems) return target.marks
    const readItem = createMarks(
      problems,
      getDeck(id).cards.map((card) => card.id),
      [],
    )
    // Marks equal to the previous marks keep the displayed marks unchanged.
    const marks = target && areMarksEqual(target.marks, readItem) ? target.marks : readItem
    marked.set(id, { problems, marks })
    return marks
  }

  /**
   * A deck the tab itself turned, which is what it now holds. It is put here
   * as it goes into the store, so the string coming back is not read again.
   */
  const setParsed = (id: string, body: string, deck: BufferDeck): void => {
    parsed.set(id, { body, deck })
  }

  /** What a tab that has closed was read as. */
  const forgetTab = (id: string): void => {
    parsed.delete(id)
    marked.delete(id)
  }

  return { getDeck, getMarks, setParsed, forgetTab }
}
