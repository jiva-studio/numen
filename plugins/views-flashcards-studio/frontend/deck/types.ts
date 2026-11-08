/**
 * Type declarations for the flashcards deck tab domain.
 */
import type { ComputedRef } from 'vue'
import type { OpenNote } from '../buffer'
import type { Marks, Surrounds, Value } from '../cards'
import type { DeckCard, DeckSection, CardModel } from '../ui/cards'
import type { Choice, DeckPreset } from './model/useDeckSchedule'

export type { Choice, DeckPreset }

/**
 * One card as the editor holds it: the file's content, addressed by its identity.
 */
export interface BufferCard {
  readonly id: string
  readonly cardId: string
  readonly section: string | null
  readonly heading: string
  readonly cardModelLink: string
  readonly cardModelPath: string
  readonly preamble: string
  readonly values: readonly Value[]
}

/** One section as the window holds it, under an identity of its own. */
export interface BufferSection {
  readonly id: string
  readonly name: string
  readonly preamble: string
}

/** A deck as the window holds it. */
export interface BufferDeck extends Surrounds {
  readonly cards: readonly BufferCard[]
  readonly sections: readonly BufferSection[]
}

/** A deck of no cards, which is what a file nothing has been written to holds. */
export const NO_DECK: BufferDeck = { preamble: '', cards: [], sections: [], tail: '' }

/** Reactive data state of an open deck tab. */
export interface DeckDataState {
  readonly id: string
  readonly note: ComputedRef<OpenNote>
  readonly deck: ComputedRef<BufferDeck>
  readonly deckCards: ComputedRef<readonly DeckCard[]>
  readonly sections: ComputedRef<readonly DeckSection[]>
  readonly cardModels: ComputedRef<readonly CardModel[]>
  readonly marks: ComputedRef<Marks>
  readonly errorMessage: ComputedRef<string>
}

/** Schedule preset state for a deck tab. */
export interface DeckScheduleState {
  readonly scheduled: ComputedRef<DeckPreset>
  readonly choices: ComputedRef<readonly Choice[]>
  setSchedule(preset: string): void
}

/** Actions and mutations available on a deck tab. */
export interface DeckActions {
  addCard(
    cardModel: string,
    values: readonly { field: string; text: string }[],
    section: string | null,
  ): void
  removeCard(card: string): void
  moveCard(card: string, target: string | null): void
  writeCardField(card: string, field: string, index: number, text: string): void
  addSection(name: string): void
  renameSection(section: string, name: string): void
  removeSection(section: string): void
  keepMine(): void
  takeFile(): void
  close(id: string): void
}

/** Composite state of a deck tab. */
export type DeckTabState = DeckDataState & DeckScheduleState & DeckActions
