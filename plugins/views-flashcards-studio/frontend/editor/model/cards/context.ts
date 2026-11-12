/**
 * What every card of one deck editor shares, however deeply it is nested. A value
 * is written four components below the editor, which reports the change to its
 * caller.
 */
import type { InjectionKey } from 'vue'

interface CardFieldTarget {
  readonly field: string
  readonly index: number
}

export interface DeckContext {
  readonly write: (card: string, target: CardFieldTarget, text: string) => void
}

export const DECK_CONTEXT = Symbol('deck') as InjectionKey<DeckContext>
