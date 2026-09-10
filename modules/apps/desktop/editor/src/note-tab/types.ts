/**
 * Types and interfaces for note tabs.
 */
import type { ComputedRef } from 'vue'
import type { Change } from './drawing'
import type { OpenNote } from './noteTypes'
import type { NoteTitlesDeps } from './titles'

export type { NoteTitlesDeps }

/** What the notes of a window ask of the vault. */
export interface NoteTabDeps extends NoteTitlesDeps {
  resolve(from: string, written: readonly string[]): Promise<ReadonlyMap<string, string>>
}

/** State and operations for an open note tab. */
export interface NoteTabState {
  readonly id: string
  readonly shown: ComputedRef<OpenNote>
  readonly saying: ComputedRef<string>
  readonly change: ComputedRef<Change | null>
  updateBody(body: string): void
  typed(body: string): void
  save(): void
  keepMine(): void
  keep(): void
  takeFile(): void
  take(): void
  setEditor(editor: unknown): void
  drew(editor: unknown): void
  measure(): void
  followLink(address: string): void
  follows(address: string): void
  close(id: string): void
  shuts(id: string): void
}
