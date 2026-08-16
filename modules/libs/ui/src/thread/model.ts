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
  readonly state?: TurnState
}

/** A turn with everything about how to draw it worked out. */
export interface PlacedTurn {
  readonly turn: Turn
  readonly voice: VoiceDescriptor
  readonly state: TurnState
  /** Where the caret goes: still arriving, and nothing follows it. */
  readonly caret: boolean
}

/**
 * The turns, with what is true of each where it sits.
 *
 * Only the last turn carries the caret.
 */
export const placeTurns = (turns: readonly Turn[]): readonly PlacedTurn[] =>
  turns.map((turn, index) => {
    const state = turn.state ?? 'settled'
    return {
      turn,
      voice: VOICES[turn.voice],
      state,
      caret: state === 'arriving' && index === turns.length - 1,
    }
  })
