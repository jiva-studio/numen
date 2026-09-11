/**
 * Type declarations for the flashcards stencil tab domain.
 */
import type { ComputedRef } from 'vue'
import type { Half } from '@numen/ui'
import type { OpenNote } from '../note-editor/notes'
import type { Marks } from '../../entities/deck/marks'
import type { BufferStencil } from './stencil'

export interface StencilDataState {
  readonly id: string
  readonly shown: ComputedRef<OpenNote>
  readonly stencil: ComputedRef<BufferStencil>
  readonly marks: ComputedRef<Marks>
  readonly errorMessage: ComputedRef<string>
}

export interface StencilFieldActions {
  addField(name: string): void
  renameField(field: string, name: string): void
  removeField(field: string): void
  moveField(field: string, at: string | null): void
}

export interface StencilFaceActions {
  addFace(name: string): void
  renameFace(id: string, name: string): void
  removeFace(id: string): void
  moveFace(id: string, at: string | null): void
  writeFaceHalf(id: string, half: Half, text: string): void
}

export interface StencilTabLifecycle {
  keepMine(): void
  takeFile(): void
  close(id: string): void
}

/** What one stencil tab holds. */
export type StencilTabState = StencilDataState &
  StencilFieldActions &
  StencilFaceActions &
  StencilTabLifecycle
