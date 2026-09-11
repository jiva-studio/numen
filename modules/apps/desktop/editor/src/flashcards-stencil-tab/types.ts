/**
 * Type declarations for the flashcards stencil tab domain.
 */
import type { ComputedRef } from 'vue'
import type { Half } from '@numen/ui'
import type { OpenNote } from '../note-tab/notes'
import type { Marks } from '../shared/flashcards/marks'
import type { BufferStencil } from './stencil'

/** What one stencil tab holds. */
export interface StencilTabState {
  /** The identity this stencil opened under, which its tab keeps wherever it goes. */
  readonly id: string
  /** The stencil as the window draws it: the state it is in, and what it stands at. */
  readonly shown: ComputedRef<OpenNote>
  /** The fields and the faces, as the editor draws them. */
  readonly stencil: ComputedRef<BufferStencil>
  /** What is wrong with the file, against the face or the field it stands on. */
  readonly marks: ComputedRef<Marks>
  /** What the whole file was refused for, in words a person reads. */
  readonly errorMessage: ComputedRef<string>
  addField(name: string): void
  renameField(field: string, name: string): void
  removeField(field: string): void
  moveField(field: string, at: string | null): void
  addFace(name: string): void
  renameFace(id: string, name: string): void
  removeFace(id: string): void
  moveFace(id: string, at: string | null): void
  writeFaceHalf(id: string, half: Half, text: string): void
  keepMine(): void
  takeFile(): void
  close(id: string): void
}
