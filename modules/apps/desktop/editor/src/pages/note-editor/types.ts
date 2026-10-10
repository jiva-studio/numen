import type { ComputedRef } from 'vue'
import type { Extension } from '@numen/ui'
import type { Change } from './model/hold'
import type { CreateResult } from '@/entities/file'
import type { NewNote, NoteHeading, OpenNote } from '@/entities/note'
import type { SearchDeps } from '@/features/command-palette'
import type { NoteTitlesDeps } from './model/titles'

export type { NoteTitlesDeps }

/** What the notes of a window ask of the vault. */
export interface NoteTabDeps extends NoteTitlesDeps, Pick<SearchDeps, 'names'> {
  resolve(from: string, written: readonly string[]): Promise<ReadonlyMap<string, string>>
  headings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  create(note: NewNote): Promise<CreateResult>
}

/** State and operations for an open note tab. */
export interface NoteTabState {
  readonly id: string
  readonly note: ComputedRef<OpenNote>
  readonly errorMessage: ComputedRef<string>
  readonly change: ComputedRef<Change | null>
  readonly extensions: Extension
  updateBody(body: string): void
  save(): void
  keepMine(): void
  takeFile(): void
  setEditor(editor: unknown): void
  measure(): void
  followLink(url: string): void
  close(id: string): void
}
