/**
 * Domain types for flashcard deck and card model open buffers.
 */
import type { Result } from '@numen/wire'

export type NoteErrorCode =
  'tooLarge' | 'bodyUnwritable' | 'notANote' | 'notText' | 'unreadable' | 'unreachable'

export type State = 'loading' | 'clean' | 'unsaved' | 'saving' | 'stale' | 'gone' | 'error'

export interface NoteBaseline {
  readonly prose?: string
  readonly fingerprint: string
}

export interface LinkAddress {
  readonly url: string
  readonly embed: string
}

export interface NoteContents {
  readonly body: string
  readonly link?: LinkAddress
  readonly at?: string
}

export type NoteResult = Result<NoteContents, string>
export type WriteResult = Result<NoteContents, string>

export interface OpenNote {
  readonly path: string
  readonly body: string
  readonly state: State
  readonly error: NoteErrorCode | null
}

export interface Notes {
  read(path: string): Promise<NoteResult>
  write(path: string, body: string, observed: NoteBaseline | null): Promise<WriteResult>
}

export interface ConflictWords {
  readonly says: string
  readonly keep: string
  readonly take: string
}

export interface OpenNotesOptions {
  limits?: { debounceMs?: number; maxWaitMs?: number }
  onReplaced?: (path: string) => void
}
