/**
 * Type declarations for the flashcards deck tab domain.
 */
import type { ComputedRef } from 'vue'
import type { DeckCard, DeckSection, Stencil } from '@numen/ui'
import type { OpenNote } from '../note-tab/notes'
import type { Marks } from '../shared/flashcards/marks'
import type { BufferDeck } from './deck'
import type { Choice, DeckPreset } from './scheduler'

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
  readonly saying: ComputedRef<string>
  /** The preset this deck is scheduled by. */
  readonly scheduled: ComputedRef<DeckPreset>
  /** The presets this deck may be put on, the defaults first. */
  readonly choices: ComputedRef<readonly Choice[]>
  schedules(preset: string): void
  setSchedule(preset: string): void
  adds(
    stencil: string,
    values: readonly { field: string; text: string }[],
    section: string | null,
  ): void
  addCard(
    stencil: string,
    values: readonly { field: string; text: string }[],
    section: string | null,
  ): void
  removes(card: string): void
  removeCard(card: string): void
  moves(card: string, at: string | null): void
  moveCard(card: string, at: string | null): void
  writes(card: string, field: string, nth: number, text: string): void
  writeCardField(card: string, field: string, nth: number, text: string): void
  addsSection(name: string): void
  addSection(name: string): void
  namesSection(section: string, name: string): void
  renameSection(section: string, name: string): void
  removesSection(section: string): void
  removeSection(section: string): void
  keep(): void
  keepMine(): void
  take(): void
  takeFile(): void
  shuts(id: string): void
  close(id: string): void
}
