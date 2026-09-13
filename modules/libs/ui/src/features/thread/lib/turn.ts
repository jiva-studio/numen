/**
 * What a thread is made of, as plain values.
 */

import { groupDigits } from '@/shared/lib/digits'

export interface VoiceDescriptor {
  /** Drawn in a bubble of its own, or as text on the surface. */
  readonly bubble: boolean
  /** Which edge the turn sits against. */
  readonly against: 'start' | 'end'
}

/**
 * Every voice a turn can be in, declared once. The shape, the side and the
 * colours all read from here.
 */
export const VOICES = {
  asked: { bubble: true, against: 'end' },
  answered: { bubble: false, against: 'start' },
  doing: { bubble: false, against: 'start' },
} as const satisfies Record<string, VoiceDescriptor>

export type Voice = keyof typeof VOICES

/** How far along a turn is. Settled unless it says otherwise. */
export type TurnState = 'settled' | 'arriving' | 'failed'

export interface Turn {
  /** Whatever the caller addresses this turn by. Never read, only handed back. */
  readonly id: string
  readonly voice: Voice
  readonly text: string
  /** What the turn is about, for a voice that has something to be about. */
  readonly about?: string
  /**
   * What is true of the turn beside what it says: how much of a call has been
   * written, how long a wait has lasted, what a finished piece of work took.
   * It is the part that moves while nothing else does.
   */
  readonly aside?: string
  readonly state?: TurnState
  /** Whether the turn stands for somewhere the person can be taken. */
  readonly opens?: boolean
  /**
   * The addresses this turn points at that reach nothing. A link carrying one
   * is drawn as not resolving.
   */
  readonly unresolved?: readonly string[]
}

/** A turn with everything about how to draw it worked out. */
export interface PlacedTurn {
  readonly turn: Turn
  readonly voice: VoiceDescriptor
  readonly state: TurnState
}

/** The turns, with what is true of each where it sits. */
export const placeTurns = (turns: readonly Turn[]): readonly PlacedTurn[] =>
  turns.map((turn) => {
    const state = turn.state ?? 'settled'
    return {
      turn,
      voice: VOICES[turn.voice],
      state,
    }
  })

/**
 * How much of a call has been written, in words.
 *
 * A call carrying the text of a note is written for minutes, and this is the
 * only thing about it that moves. Grouped in thousands, because the numbers
 * reach five figures on one note.
 */
export const writeCharCount = (count: number): string =>
  count <= 0 ? '' : `${groupDigits(count)} characters`
