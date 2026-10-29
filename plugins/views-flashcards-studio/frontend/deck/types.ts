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
  /** The identity the window addresses it by: its card id, or one minted for it. */
  readonly id: string
  /**
   * What the card is, for as long as it exists, without the caret its heading
   * writes it behind. Empty for a card the application has not written yet.
   */
  readonly cardId: string
  /**
   * The section the card is placed under, by the identity the editor knows that
   * section by. Null for a card before the first section.
   */
  readonly section: string | null
  /**
   * The heading line, with the card id removed.
   */
  readonly heading: string
  /**
   * The card model it is cut by, as the wikilink beneath its heading names it.
   */
  readonly cardModelLink: string
  /**
   * Where that card model is filed, as the wikilink resolved in the vault.
   */
  readonly cardModelPath: string
  /** The prose between that wikilink and the first field. */
  readonly preamble: string
  readonly values: readonly Value[]
}

/** One section as the window holds it, under an identity of its own. */
export interface BufferSection {
  /** The identity the window addresses it by. */
  readonly id: string
  /** What it is called, as its heading spells it. Two sections may carry one name. */
  readonly name: string
  /** The prose between its heading and its first card. */
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
