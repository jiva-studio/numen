/**
 * Type declarations for the flashcards deck tab domain.
 */
import type { ComputedRef } from 'vue'
import type { DeckCard, DeckSection, Stencil } from '@numen/ui'
import type { OpenNote } from '../note-editor/notes'
import type { Marks } from '../../shared/flashcards/marks'
import type { Surrounds } from '../../shared/flashcards/surrounds'
import type { Value } from '../../shared/flashcards/cards'
import type { Choice, DeckPreset } from './scheduler'

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
   * The line its heading says, with the mark taken off. Nothing draws it and
   * nothing is typed into it: a write reads it again from the first field.
   */
  readonly heading: string
  /**
   * The stencil it is cut by, as the wikilink beneath its heading names it: a
   * name where one picks the stencil, and `note://<identifier>` where none does.
   */
  readonly stencilLink: string
  /**
   * Where that stencil is filed, as the wikilink resolved in the vault. Empty
   * for a card naming none and for a name that reaches no note.
   */
  readonly stencilPath: string
  /** The prose between that wikilink and the first field. */
  readonly preamble: string
  readonly values: readonly Value[]
}

/** One section as the window holds it, under an identity of its own. */
export interface BufferSection {
  /** The identity the window addresses it by, minted at every reading. */
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

/** What one deck tab holds. */
export interface DeckTabState {
  /** The identity this deck opened under, which its tab keeps wherever it goes. */
  readonly id: string
  /** The deck as the window draws it: the state it is in, and what it stands at. */
  readonly shown: ComputedRef<OpenNote>
  /** The cards, as the window holds them. */
  readonly deck: ComputedRef<BufferDeck>
  /** The same, as the grid draws them, each under the stencil that cuts it. */
  readonly drawn: ComputedRef<readonly DeckCard[]>
  /** The sections, as the grid draws them. */
  readonly sections: ComputedRef<readonly DeckSection[]>
  /** The stencils a card may be cut by. */
  readonly stencils: ComputedRef<readonly Stencil[]>
  /** What is wrong with the file, against the card it stands on. */
  readonly marks: ComputedRef<Marks>
  /** What the whole file was refused for, in words a person reads. */
  readonly errorMessage: ComputedRef<string>
  /** The preset this deck is scheduled by. */
  readonly scheduled: ComputedRef<DeckPreset>
  /** The presets this deck may be put on, the defaults first. */
  readonly choices: ComputedRef<readonly Choice[]>
  setSchedule(preset: string): void
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
