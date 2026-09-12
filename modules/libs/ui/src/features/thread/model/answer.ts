/**
 * The answer as it arrives: put up with its first words, grown a frame at a
 * time, and settled when it ends. An answer with nothing in it leaves no turn.
 */
import type { Turn } from '../lib/turn'

/**
 * When the words that have arrived are put on the screen.
 *
 * A model writes faster than a screen draws, and every piece put up on its own
 * marks up the whole answer again. They are collected and put up once a frame,
 * which is as often as anybody can see.
 */
export type Paint = (draw: () => void) => void

export interface Answer {
  /** More words of the answer. */
  readonly say: (text: string) => void
  /** The answer ends where it stands, and the next words begin another. */
  readonly settle: () => void
  /** Whether a turn is open for the words still arriving. */
  readonly isOpen: () => boolean
  /** Whether the last answer carried any words. */
  readonly isSaid: () => boolean
}

/** One growing answer, under names taken from `nextId` as each turn begins. */
export function createAnswer(
  put: (turn: Turn) => void,
  drop: (id: string) => void,
  nextId: () => string,
  paint: Paint,
): Answer {
  let id = ''
  let text = ''

  // The words collected since the last frame, and whether one is coming.
  let painting = false

  const show = () => {
    if (painting || !id) return
    painting = true
    paint(() => {
      painting = false
      if (id) put({ id, voice: 'answered', text, state: 'arriving' })
    })
  }

  return {
    say: (words: string) => {
      if (!id) {
        id = nextId()
        text = words
        // The turn goes up with its first words in the same step the wait comes
        // down. A window nobody is looking at draws no frames, and the frames
        // are what grow the turn.
        put({ id, voice: 'answered', text, state: 'arriving' })
        return
      }
      text += words
      show()
    },
    settle: () => {
      if (!id) return
      // An answer with nothing in it is a turn with no words and a gap either
      // side of it.
      if (text === '') drop(id)
      else put({ id, voice: 'answered', text })
      id = ''
    },
    isOpen: () => id !== '',
    isSaid: () => text !== '',
  }
}
