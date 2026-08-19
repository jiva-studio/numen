/**
 * What the foot of the window says about work nobody asked for.
 *
 * A vault reads itself in phases that count different things: books opened,
 * then windows of text embedded. Which phase is running, which count goes with
 * it, and how much is left are rules, and they are here so that a test can ask
 * them without a screen.
 *
 * No wording: this answers with a phase, and the window keeps the sentences.
 */
import type { Tally } from '@numen/ui'

/** What the vault is doing. */
export type Phase =
  /** Reading the vault: its notes, then its books. */
  | 'reading'
  /** Turning what was cut into vectors. */
  | 'learning'
  /** Nothing is coming: this installation has no model, and words are the whole search. */
  | 'wordsOnly'
  /** Nothing to say. */
  | 'idle'

/** What the vault says about itself, as the foot of the window reads it. */
export interface Reading {
  /** Whether the vault has work in hand. It says so; the counts do not. */
  readonly busy: boolean
  /** Whether that work is embedding, which counts chunks and not books. */
  readonly learning: boolean
  /** The source open now, empty between sources and after the last. */
  readonly reading: string
  readonly books: number
  readonly booksRead: number
  readonly chunks: number
  readonly embedded: number
  /** What the pass now running found to do, and how much of it is done. */
  readonly owing: number
  readonly made: number
  /** Whether anything is going to embed what was cut. */
  readonly embedding: boolean
  /** How fast the count of the phase now running is moving, a second. */
  readonly rate: number
}

/** What the foot draws. */
export interface Foot {
  readonly phase: Phase
  /** What the work is on, when that is worth saying. */
  readonly about: string
  /** Whether the work named is happening now. */
  readonly working: boolean
  /** Where it has got to, when there is a total to draw against. */
  readonly tally?: Tally
  /** How much of this phase is left, and how fast it is going. */
  readonly left: number
  readonly perSecond: number
}

const quiet: Foot = { phase: 'idle', about: '', working: false, left: 0, perSecond: 0 }

/**
 * What the foot of the window says.
 *
 * The vault's own word for having work in hand is what decides that there is
 * something to say. Reading a book and embedding one change no file, and cutting
 * a library begins after the notes are read, so a count of nothing is what the
 * work looks like both before it starts and while it runs.
 *
 * A tally is drawn only where there is a total to draw against. The first
 * seconds of a scan have no total and are still work.
 */
export const footOf = (v: Reading): Foot => {
  if (v.busy && v.learning) {
    return {
      phase: 'learning',
      about: v.reading,
      working: true,
      ...(v.owing > 0 ? { tally: { done: v.made, total: v.owing } } : {}),
      left: Math.max(0, v.owing - v.made),
      perSecond: v.rate,
    }
  }
  if (v.busy) {
    return {
      phase: 'reading',
      about: v.reading,
      working: true,
      ...(v.books > 0 ? { tally: { done: v.booksRead, total: v.books } } : {}),
      left: Math.max(0, v.books - v.booksRead),
      perSecond: v.rate,
    }
  }
  // Half the search is missing and nothing is going to bring it: a fact about
  // this installation, said once and quietly.
  if (v.chunks > 0 && !v.embedding) {
    return { phase: 'wordsOnly', about: '', working: false, left: 0, perSecond: 0 }
  }
  return quiet
}
