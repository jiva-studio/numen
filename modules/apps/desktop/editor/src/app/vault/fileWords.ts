/**
 * File domain conversions between schema representations and window models.
 */
import {
  BookFormat as BookFormats,
  SourceKind,
} from '@numen/protocol'
import type {
  Entry as EntryMessage,
  MoveResult as MoveResultMessage,
} from '@numen/protocol'
import type { BookFormat, DocumentFormat, Entry, MoveResult, Source } from '@/entities/file'
import { noteType } from './noteWords'

/** A source this window has no word for is a file it holds no source for. */
const holding: Record<SourceKind, Source> = {
  [SourceKind.UNSPECIFIED]: 'other',
  [SourceKind.NOTE]: 'note',
  [SourceKind.BOOK]: 'book',
  [SourceKind.RECORDING]: 'recording',
  [SourceKind.URL]: 'url',
}

export const sourceKind = (of: SourceKind): Source => holding[of] ?? 'other'

/** How a book at a path is drawn, in the words the window uses. */
const drawn: Record<BookFormats, BookFormat | DocumentFormat | undefined> = {
  [BookFormats.UNSPECIFIED]: undefined,
  [BookFormats.PDF]: 'pdf',
  [BookFormats.EPUB]: 'epub',
}

export const bookFormat = (of: BookFormats): BookFormat | DocumentFormat | undefined => drawn[of]

/** One row of a listing, kept as the plain value the window carries it as. */
export const mapEntry = (one: EntryMessage): Entry => ({
  path: one.path,
  name: one.name,
  folder: one.folder,
  kind: sourceKind(one.kind),
  type: noteType(one.type),
})

/** What the file did, in the shape the window carries it. */
export const mapMoveResult = (answer: MoveResultMessage): MoveResult => ({
  from: answer.from,
  to: answer.to,
  repaired: answer.repaired,
})
