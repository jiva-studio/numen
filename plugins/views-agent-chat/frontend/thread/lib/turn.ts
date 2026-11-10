/**
 * What a thread is made of, as plain values.
 */

import { groupDigits, type LinkPiece } from '@numen/ui'
import type { Turn, TurnState, Voice } from '../../types'

export interface VoiceDescriptor {
  readonly isBubble: boolean
  readonly against: 'start' | 'end'
}

/**
 * Every voice a turn can be in, declared once. The shape, the side and the
 * colours all read from here.
 */
export const VOICES = {
  prompt: { isBubble: true, against: 'end' },
  response: { isBubble: false, against: 'start' },
  doing: { isBubble: false, against: 'start' },
} as const satisfies Record<Voice, VoiceDescriptor>

/** A turn with everything about how to draw it worked out. */
export interface PlacedTurn {
  readonly turn: Turn
  readonly voice: VoiceDescriptor
  readonly state: TurnState
}

/** The unresolved addresses of a turn that names none. */
export const NO_UNRESOLVED: readonly string[] = Object.freeze([])

/** The links of a turn that has none. */
export const NO_LINKS: readonly LinkPiece[] = Object.freeze([])

const placedByTurn = new WeakMap<Turn, PlacedTurn>()

const placeTurn = (turn: Turn): PlacedTurn => {
  const recognized = placedByTurn.get(turn)
  if (recognized) return recognized
  const placedTurn: PlacedTurn = {
    turn,
    voice: VOICES[turn.voice],
    state: turn.state ?? 'settled',
  }
  placedByTurn.set(turn, placedTurn)
  return placedTurn
}

/** The turns, with what is true of each where it is located. A turn keeps the entry it was first given. */
export const placeTurns = (turns: readonly Turn[]): readonly PlacedTurn[] => turns.map(placeTurn)

/**
 * How much of a call has been written, in words.
 *
 * A call carrying the text of a note is written for minutes, and this is the
 * only thing about it that moves. Grouped in thousands, because the numbers
 * reach five figures on one note.
 */
export const writeCharCount = (count: number): string =>
  count <= 0 ? '' : `${groupDigits(count)} characters`
