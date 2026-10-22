/**
 * What every card of one deck editor shares, however deep it stands. A value is
 * written four components below the editor that tells the caller of it.
 */
import type { InjectionKey } from 'vue'

export interface CardFieldTarget {
  readonly field: string
  readonly index: number
}

export interface DeckContext {
  /** One value of a card as it now reads; `target` says which of a field's values. */
  readonly write: (card: string, target: CardFieldTarget, text: string) => void
}

export const DECK_CONTEXT = Symbol('deck') as InjectionKey<DeckContext>
