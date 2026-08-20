/**
 * What the corner of the window draws.
 *
 * The application says what it is doing, and each piece of it is one card. A
 * new kind of work is an entry in that list and nothing here.
 *
 * The one card that is not work is the one saying this installation will not
 * embed what it cut. It is a rule, and it is here so that a test can ask it
 * without a screen.
 */
import type { Notice } from '@numen/ui'
import type { Task } from './showing'

/** What the vault says about itself that the corner has anything to say about. */
export interface Reading {
  /** Spans of text the index holds. */
  readonly chunks: number
  /** Whether anything is going to turn the chunks into vectors. */
  readonly embedding: boolean
}

/** The sentences the corner draws that are the window's own. */
export interface Words {
  /** One way of asking is missing and nothing is going to bring it. */
  readonly words: string
}

/**
 * The cards the corner draws.
 *
 * One piece of work is one card. Two accounts of one piece of work are two
 * cards with two counts, and a person cannot tell which of them is theirs.
 */
export const cornerOf = (
  tasks: readonly Task[],
  vault: Reading,
  words: Words,
): readonly Notice[] => {
  const out: Notice[] = tasks.map((at) => ({
    id: at.id,
    says: at.doing,
    about: at.about,
    working: !at.failed,
    trouble: at.failed,
    asked: at.asked,
    left: '',
    ...(at.total > 0 ? { done: at.done, total: at.total } : {}),
  }))

  // Said once and quietly, and it is so whether or not anything is running.
  if (vault.chunks > 0 && !vault.embedding) {
    out.push({ id: 'wordsOnly', says: words.words, about: '', working: false, left: '' })
  }
  return out
}
