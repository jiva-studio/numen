/**
 * File domain conversions between schema representations and window models.
 */
import type { Entry as EntryMessage, MoveResult as MoveResultMessage } from '@numen/protocol'
import type { Entry, MoveResult } from '@/entities/file'
import { mapNoteType } from './noteWords'

/** One row of a listing, kept as the plain value the window carries it as. */
export const mapEntry = (one: EntryMessage): Entry => ({
  path: one.path,
  name: one.name,
  isFolder: one.isFolder,
  isNote: one.isNote,
  type: mapNoteType(one.type),
})

/** What the file did, in the shape the window carries it. */
export const mapMoveResult = (answer: MoveResultMessage): MoveResult => ({
  from: answer.from,
  to: answer.to,
  repaired: answer.repaired,
})
