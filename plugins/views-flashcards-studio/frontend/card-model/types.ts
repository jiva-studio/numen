/**
 * Type declarations for the flashcards card model tab domain.
 */
import type { ComputedRef } from 'vue'
import type { Half } from '../ui/cards'
import type { OpenNote } from '../buffer'
import type { Marks } from '../cards'
import type { BufferCardModel } from './lib/card-model'

export interface CardModelDataState {
  readonly id: string
  readonly note: ComputedRef<OpenNote>
  readonly cardModel: ComputedRef<BufferCardModel>
  readonly marks: ComputedRef<Marks>
  readonly errorMessage: ComputedRef<string>
}

export interface CardModelFieldActions {
  addField(name: string): void
  renameField(field: string, name: string): void
  removeField(field: string): void
  moveField(field: string, target: string | null): void
}

export interface CardModelTemplateActions {
  addTemplate(name: string): void
  renameTemplate(id: string, name: string): void
  removeTemplate(id: string): void
  moveTemplate(id: string, target: string | null): void
  writeTemplateHalf(id: string, half: Half, text: string): void
}

export interface CardModelTabLifecycle {
  keepMine(): void
  takeFile(): void
  close(id: string): void
}

/** What one card model tab holds. */
export type CardModelTabState = CardModelDataState &
  CardModelFieldActions &
  CardModelTemplateActions &
  CardModelTabLifecycle
