/**
 * The answer as it arrives: put up with its first words, grown a frame at a
 * time, and settled when it ends. An answer with nothing in it leaves no turn.
 */
import type { Turn } from '../lib/turn'
import type { Paint } from '@/shared/lib/time/clock'

export interface Answer {
  /** More words of the answer. */
  readonly say: (text: string) => void
  /** The answer ends at its current position, and the next words begin another. */
  readonly settle: () => void
  /** Mark the current answering turn as failed. */
  readonly fail: () => void
  /** The accumulated text of the current answer. */
  readonly getText: () => string
  /** Whether a turn is open for the words still arriving. */
  readonly isOpen: () => boolean
  /** Whether the last answer carried any words. */
  readonly hasContent: () => boolean
}

export interface AnswerActions {
  readonly put: (turn: Turn) => void
  readonly drop: (id: string) => void
  readonly nextId: () => string
}

/** One growing answer, under names taken from `nextId` as each turn begins. */
export function createAnswer(actions: AnswerActions, paint: Paint): Answer {
  const { put, drop, nextId } = actions
  let id = ''
  let text = ''

  // The words collected since the last frame, and whether one is coming.
  let painting = false

  const show = () => {
    if (painting || !id) return
    painting = true
    paint(() => {
      painting = false
      if (id) put({ id, voice: 'response', text, state: 'arriving' })
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
        put({ id, voice: 'response', text, state: 'arriving' })
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
      else put({ id, voice: 'response', text })
      id = ''
    },
    fail: () => {
      if (!id) return
      if (text === '') drop(id)
      else put({ id, voice: 'response', text, state: 'failed' })
      id = ''
    },
    getText: () => text,
    isOpen: () => id !== '',
    hasContent: () => text !== '',
  }
}
