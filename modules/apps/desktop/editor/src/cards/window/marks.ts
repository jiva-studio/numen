/**
 * Where each problem a reading came back with is drawn: against a card,
 * against a face, against one field of a card, or against the file itself.
 *
 * A problem says where it stands, and the window draws it on the thing
 * standing there. Two readings wrong in the same way leave what is drawn where
 * it is, so a mark a person is reading does not blink on every read.
 */
import type { Problem } from '../vault/cards'

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
