/**
 * File domain conversions between schema representations and window models.
 */
import { SourceKind } from '@numen/protocol'
import type { Entry as EntryMessage, MoveResult as MoveResultMessage } from '@numen/protocol'
import type { Entry, MoveResult } from '@/entities/file'
import { mapNoteType } from './noteWords'

/** Whether the vault holds a note at a path. */
export const isNoteKind = (of: SourceKind): boolean => of === SourceKind.NOTE

/** One row of a listing, kept as the plain value the window carries it as. */
export const mapEntry = (one: EntryMessage): Entry => ({
  path: one.path,
  name: one.name,
  isFolder: one.isFolder,
  isNote: isNoteKind(one.kind),
  type: mapNoteType(one.type),
})

/** What the file did, in the shape the window carries it. */
export const mapMoveResult = (answer: MoveResultMessage): MoveResult => ({
  from: answer.from,
  to: answer.to,
  repaired: answer.repaired,
})
