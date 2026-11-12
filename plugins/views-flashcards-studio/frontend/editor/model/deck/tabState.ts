/**
 * The state of an open deck tab.
 */
import type { ComputedRef } from 'vue'
import type { OpenNote } from '../../lib/buffer/types'
import type { Marks } from '../../../lib/marks'
import type { DeckCard, DeckSection, CardModel } from '../../../lib/cards'
import type { BufferDeck } from '../../lib/deck/types'
import type { Choice, DeckPreset } from './useDeckSchedule'

/** Reactive data state of an open deck tab. */
interface DeckDataState {
  readonly note: ComputedRef<OpenNote>
  readonly deck: ComputedRef<BufferDeck>
  readonly deckCards: ComputedRef<readonly DeckCard[]>
  readonly sections: ComputedRef<readonly DeckSection[]>
  readonly cardModels: ComputedRef<readonly CardModel[]>
  readonly marks: ComputedRef<Marks>
  readonly errorMessage: ComputedRef<string>
}

interface DeckScheduleState {
  readonly scheduled: ComputedRef<DeckPreset>
  readonly choices: ComputedRef<readonly Choice[]>
  setSchedule(preset: string): void
}

/** Edits to the cards and sections of a deck tab. */
interface DeckActions {
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
}

/** What the tab does with the file it shows. */
interface DeckFileActions {
  keepMine(): void
  takeFile(): void
  close(id: string): void
}

/** The state of a deck tab, in named parts. */
export interface DeckTabState {
  readonly id: string
  readonly data: DeckDataState
  readonly schedule: DeckScheduleState
  readonly actions: DeckActions
  readonly file: DeckFileActions
}
