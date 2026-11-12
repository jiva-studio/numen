/**
 * Type declarations for the flashcards card model tab domain.
 */
import type { ComputedRef } from 'vue'
import type { Half } from '../../../lib/cards'
import type { OpenNote } from '../../lib/buffer/types'
import type { Marks } from '../../../lib/marks'
import type { BufferCardModel } from '../../lib/card-model/card-model'

interface CardModelDataState {
  readonly note: ComputedRef<OpenNote>
  readonly cardModel: ComputedRef<BufferCardModel>
  readonly marks: ComputedRef<Marks>
  readonly errorMessage: ComputedRef<string>
}

interface CardModelFieldActions {
  addField(name: string): void
  renameField(field: string, name: string): void
  removeField(field: string): void
  moveField(field: string, target: string | null): void
}

interface CardModelTemplateActions {
  addTemplate(name: string): void
  renameTemplate(id: string, name: string): void
  removeTemplate(id: string): void
  moveTemplate(id: string, target: string | null): void
  writeTemplateHalf(id: string, half: Half, text: string): void
}

interface CardModelTabLifecycle {
  keepMine(): void
  takeFile(): void
  close(id: string): void
}

/** What one card model tab holds, in named parts. */
export interface CardModelTabState {
  readonly id: string
  readonly data: CardModelDataState
  readonly fields: CardModelFieldActions
  readonly templates: CardModelTemplateActions
  readonly file: CardModelTabLifecycle
}
