/**
 * Types and interfaces for note tabs.
 */
import type { ComputedRef } from 'vue'
import type { Change } from './model/hold'
import type { OpenNote } from '@/entities/note'
import type { NoteTitlesDeps } from './model/titles'

export type { NoteTitlesDeps }

/** What the notes of a window ask of the vault. */
export interface NoteTabDeps extends NoteTitlesDeps {
  resolve(from: string, written: readonly string[]): Promise<ReadonlyMap<string, string>>
}

/** State and operations for an open note tab. */
export interface NoteTabState {
  readonly id: string
  readonly note: ComputedRef<OpenNote>
  readonly errorMessage: ComputedRef<string>
  readonly change: ComputedRef<Change | null>
  updateBody(body: string): void
  save(): void
  keepMine(): void
  takeFile(): void
  setEditor(editor: unknown): void
  measure(): void
  followLink(url: string): void
  close(id: string): void
}
