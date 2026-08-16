/**
 * What a thread is made of, as plain values.
 */

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

export const VOICE_NAMES = Object.keys(VOICES) as readonly Voice[]

/** How far along a turn is. Settled unless it says otherwise. */
export type TurnState = 'settled' | 'arriving' | 'failed'

export interface Turn {
  /** Whatever the caller addresses this turn by. Never read, only handed back. */
  readonly id: string
  readonly voice: Voice
  readonly text: string
  /** What the turn is about, for a voice that has something to be about. */
  readonly about?: string
  readonly state?: TurnState
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
