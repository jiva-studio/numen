import type { DeckProblem } from '../types'

/** Where each problem is drawn. */
export interface Marks {
  readonly at: ReadonlyMap<string, readonly string[]>
  readonly bySection: ReadonlyMap<string, ReadonlyMap<string, readonly string[]>>
  readonly fields: ReadonlyMap<string, readonly string[]>
  readonly whole: readonly string[]
}

/**
 * Each problem against the item it concerns. A problem carries its location, so
 * a card under no card model and two cards with one card id each land on the
 * tile they were read from, and a problem naming a field lands on that field of
 * that card. A problem located past the end of what was read is against the file.
 */
export const createMarks = (
  problems: readonly DeckProblem[],
  cards: readonly string[],
  templates: readonly string[],
): Marks => {
  const marksAt = new Map<string, string[]>()
  const underMap = new Map<string, Map<string, string[]>>()
  const fields = new Map<string, string[]>()
  const whole: string[] = []

  const addProblemMark = (marks: Map<string, string[]>, key: string, text: string): void => {
    const existing = marks.get(key)
    if (existing) existing.push(text)
    else marks.set(key, [text])
  }

  for (const problem of problems) {
    const card = cards[problem.card ?? -1]
    if (card !== undefined) {
      if (problem.field === '') {
        addProblemMark(marksAt, card, problem.text)
        continue
      }
      let cardMarks = underMap.get(card)
      if (!cardMarks) {
        cardMarks = new Map<string, string[]>()
        underMap.set(card, cardMarks)
      }
      addProblemMark(cardMarks, problem.field, problem.text)
      continue
    }
    const template = templates[problem.template ?? -1]
    if (template !== undefined) {
      addProblemMark(marksAt, template, problem.text)
      continue
    }
    if (problem.field !== '') {
      addProblemMark(fields, problem.field, problem.text)
      continue
    }
    whole.push(problem.text)
  }

  return { at: marksAt, bySection: underMap, fields, whole }
}

const areWordsEqual = (one: readonly string[], other: readonly string[]): boolean =>
  one.length === other.length && one.every((text, index) => text === other[index])

const areMarkMapsEqual = (
  one: ReadonlyMap<string, readonly string[]>,
  other: ReadonlyMap<string, readonly string[]>,
): boolean => {
  if (one.size !== other.size) return false
  for (const [key, texts] of one) {
    const otherVal = other.get(key)
    if (!otherVal || !areWordsEqual(texts, otherVal)) return false
  }
  return true
}

const areUnderMapsEqual = (
  one: ReadonlyMap<string, ReadonlyMap<string, readonly string[]>>,
  other: ReadonlyMap<string, ReadonlyMap<string, readonly string[]>>,
): boolean => {
  if (one.size !== other.size) return false
  for (const [key, byField] of one) {
    const otherVal = other.get(key)
    if (!otherVal || !areMarkMapsEqual(byField, otherVal)) return false
  }
  return true
}

export const areMarksEqual = (one: Marks, other: Marks): boolean =>
  areWordsEqual(one.whole, other.whole) &&
  areMarkMapsEqual(one.at, other.at) &&
  areUnderMapsEqual(one.bySection, other.bySection) &&
  areMarkMapsEqual(one.fields, other.fields)
