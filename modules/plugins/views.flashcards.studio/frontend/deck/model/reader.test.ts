/**
 * The deck one tab shows and the marks drawn on it, held against what they
 * were read from.
 */
import { describe, expect, it } from 'vitest'
import type { DeckProblem } from '../../cards'
import { serializeBufferDeckToString, type BufferCard, type BufferDeck } from '../lib/deck'
import { createDeckReader, type ShownStore } from './reader'

const card = (id: string, name: string): BufferCard => ({
  id,
  mark: '',
  section: null,
  heading: name,
  stencilLink: 'Animal',
  stencilPath: 'Animal.md',
  preamble: '',
  values: [{ field: 'Name', text: name }],
})

const createDeck = (...cards: BufferCard[]): BufferDeck => ({
  preamble: '',
  cards,
  sections: [],
  tail: '',
})

const problem = (overrides: Partial<DeckProblem>): DeckProblem => ({
  kind: 'unknown',
  card: null,
  face: null,
  field: '',
  text: 'Odd',
  ...overrides,
})

/** A store holding one string per tab, and a reader over it. */
const open = (problems: () => readonly DeckProblem[] = () => []) => {
  const bodies = new Map<string, string>()
  const store: ShownStore = {
    getOpenNote: (id) => ({ body: bodies.get(id) ?? '' }),
    getPath: (id) => `${id}.md`,
  }
  return { bodies, reader: createDeckReader(store, problems) }
}

describe('the deck a tab shows', () => {
  it('is read out of the string the store holds', () => {
    const { bodies, reader } = open()
    bodies.set('a', serializeBufferDeckToString(createDeck(card('c1', 'Llama'))))

    expect(reader.getDeck('a').cards.map((one) => one.heading)).toStrictEqual(['Llama'])
  })

  it('remains as the same deck while the string remains', () => {
    const { bodies, reader } = open()
    bodies.set('a', serializeBufferDeckToString(createDeck(card('c1', 'Llama'))))

    expect(reader.getDeck('a')).toBe(reader.getDeck('a'))
  })

  it('is the deck it was set for, under the string it was set with', () => {
    const { bodies, reader } = open()
    const body = serializeBufferDeckToString(createDeck(card('c1', 'Llama')))
    const turned = createDeck(card('c2', 'Alpaca'))
    bodies.set('a', body)

    reader.setParsed('a', body, turned)

    expect(reader.getDeck('a')).toBe(turned)
  })

  it('is read again once the tab is forgotten', () => {
    const { bodies, reader } = open()
    const body = serializeBufferDeckToString(createDeck(card('c1', 'Llama')))
    bodies.set('a', body)
    const turned = createDeck(card('c2', 'Alpaca'))
    reader.setParsed('a', body, turned)

    reader.forgetTab('a')

    expect(reader.getDeck('a')).not.toBe(turned)
    expect(reader.getDeck('a').cards.map((one) => one.id)).toStrictEqual(['c1'])
  })
})

describe('the marks on a deck', () => {
  it('stand against the card the problem counts', () => {
    const { bodies, reader } = open(() => [problem({ card: 1, text: 'Bad card' })])
    bodies.set(
      'a',
      serializeBufferDeckToString(createDeck(card('c1', 'Llama'), card('c2', 'Alpaca'))),
    )

    expect(reader.getMarks('a').at.get('c2')).toStrictEqual(['Bad card'])
    expect(reader.getMarks('a').whole).toStrictEqual([])
  })

  it('stand against the file where the problem counts a face, since no face is drawn', () => {
    const { bodies, reader } = open(() => [problem({ face: 0, text: 'Bad face' })])
    bodies.set('a', serializeBufferDeckToString(createDeck(card('c1', 'Llama'))))

    expect(reader.getMarks('a').whole).toStrictEqual(['Bad face'])
    expect(reader.getMarks('a').at.size).toBe(0)
  })

  it('stand as the same marks while the problems are the same list', () => {
    const { bodies, reader } = open(() => [problem({ card: 0 })])
    bodies.set('a', serializeBufferDeckToString(createDeck(card('c1', 'Llama'))))

    expect(reader.getMarks('a')).toBe(reader.getMarks('a'))
  })

  it('are made again once the vault reports other problems', () => {
    let problems: readonly DeckProblem[] = [problem({ card: 0, text: 'First' })]
    const { bodies, reader } = open(() => problems)
    bodies.set('a', serializeBufferDeckToString(createDeck(card('c1', 'Llama'))))
    reader.getMarks('a')

    problems = [problem({ card: 0, text: 'Second' })]

    expect(reader.getMarks('a').at.get('c1')).toStrictEqual(['Second'])
  })

  it('stand as they were where the new problems say the same', () => {
    let problems: readonly DeckProblem[] = [problem({ card: 0, text: 'Same' })]
    const { bodies, reader } = open(() => problems)
    bodies.set('a', serializeBufferDeckToString(createDeck(card('c1', 'Llama'))))
    const first = reader.getMarks('a')

    problems = [problem({ card: 0, text: 'Same' })]

    expect(reader.getMarks('a')).toBe(first)
  })

  it('are made again once the tab is forgotten', () => {
    const problems = [problem({ card: 0 })]
    const { bodies, reader } = open(() => problems)
    bodies.set('a', serializeBufferDeckToString(createDeck(card('c1', 'Llama'))))
    const first = reader.getMarks('a')

    reader.forgetTab('a')

    expect(reader.getMarks('a')).not.toBe(first)
  })
})
