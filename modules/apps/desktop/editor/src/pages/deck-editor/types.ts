/**
 * Type declarations for the flashcards deck tab domain.
 */
import type { ComputedRef } from 'vue'
import type { OpenNote } from '@/entities/note'
import type { Marks } from '@/entities/deck'
import type { Surrounds } from '@/entities/deck'
import type { Value } from '@/entities/deck'
import type { DeckCard, DeckSection, Stencil } from '@numen/ui'
import type { Choice, DeckPreset } from './model/useDeckSchedule'

export type { Choice, DeckPreset }

/**
 * One card as the window holds it: what the file says, under the identity it is
 * addressed by.
 */
export interface BufferCard {
  /** The identity the window addresses it by: its mark, or one minted for it. */
  readonly id: string
  /**
   * What the card is, for as long as it exists, without the caret its heading
   * writes it behind. Empty for a card the application has not written yet.
   */
  readonly mark: string
  /**
   * The section it stands under, by the identity this window knows that section
   * at. Nothing for a card standing before the first.
   */
  readonly section: string | null
  /**
   * The line its heading says, with the mark taken off.
   */
  readonly heading: string
  /**
   * The stencil it is cut by, as the wikilink beneath its heading names it.
   */
  readonly stencilLink: string
  /**
   * Where that stencil is filed, as the wikilink resolved in the vault.
   */
  readonly stencilPath: string
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
  readonly drawn: ComputedRef<readonly DeckCard[]>
  readonly sections: ComputedRef<readonly DeckSection[]>
  readonly stencils: ComputedRef<readonly Stencil[]>
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
    stencil: string,
    values: readonly { field: string; text: string }[],
    section: string | null,
  ): void
  removeCard(card: string): void
  moveCard(card: string, at: string | null): void
  writeCardField(card: string, field: string, nth: number, text: string): void
  addSection(name: string): void
  renameSection(section: string, name: string): void
  removeSection(section: string): void
  keepMine(): void
  takeFile(): void
  close(id: string): void
}

/** Composite state of a deck tab. */
export type DeckTabState = DeckDataState & DeckScheduleState & DeckActions
