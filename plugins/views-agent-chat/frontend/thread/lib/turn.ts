/**
 * What a thread is made of, as plain values.
 */

import { groupDigits, type LinkPiece } from '@numen/ui'
import type { Turn, TurnState } from '../../types'
import { VOICES, type VoiceDescriptor } from './voices'

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
 * How much of a call has been written, in words. Grouped in thousands, because the numbers reach five
 * figures on one note.
 */
export const writeCharCount = (count: number): string => `${groupDigits(count)} characters`
