/**
 * What the corner of the window draws.
 *
 * The application says what it is doing, and each piece of it is one card. A
 * new kind of work is an entry in that list and nothing here.
 *
 * The one card that is not work is the one saying this installation will not
 * embed what it cut.
 */
import type { Notice } from '@numen/ui'
import type { Task } from './core'
import { wordsOnly, type Meaning } from './meaning'

/** The sentences the corner draws that are the window's own. */
export interface Words {
  /** One way of asking is missing and nothing is going to bring it. */
  readonly wordsOnly: string
}

/**
 * The cards the corner draws.
 *
 * One piece of work is one card. Two accounts of one piece of work are two
 * cards with two counts, and a person cannot tell which of them is theirs.
 */
export const cornerOf = (
  tasks: readonly Task[],
  vault: Meaning,
  words: Words,
): readonly Notice[] => {
  const out: Notice[] = tasks.map((at) => ({
    id: at.id,
    says: at.doing,
    about: at.about,
    working: !at.failed,
    trouble: at.failed,
    asked: at.asked,
    ...(at.total > 0 ? { done: at.done, total: at.total, counting: at.counting } : {}),
  }))

  // Said once and quietly, and it is so whether or not anything is running.
  if (wordsOnly(vault)) {
    out.push({ id: 'wordsOnly', says: words.wordsOnly, about: '', working: false })
  }
  return out
}
