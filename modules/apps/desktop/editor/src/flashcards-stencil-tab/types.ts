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
  readonly saying: ComputedRef<string>
  addsField(name: string): void
  addField(name: string): void
  namesField(field: string, name: string): void
  renameField(field: string, name: string): void
  removesField(field: string): void
  removeField(field: string): void
  movesField(field: string, at: string | null): void
  moveField(field: string, at: string | null): void
  addsFace(name: string): void
  addFace(name: string): void
  namesFace(id: string, name: string): void
  renameFace(id: string, name: string): void
  removesFace(id: string): void
  removeFace(id: string): void
  movesFace(id: string, at: string | null): void
  moveFace(id: string, at: string | null): void
  writes(id: string, half: Half, text: string): void
  writeFaceHalf(id: string, half: Half, text: string): void
  keep(): void
  keepMine(): void
  take(): void
  takeFile(): void
  shuts(id: string): void
  close(id: string): void
}
