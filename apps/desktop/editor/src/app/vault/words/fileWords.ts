/**
 * File domain conversions between schema representations and window models.
 */
import type { Entry, MoveResult } from '@/entities/file'
import { mapNoteType } from './noteWords'

interface EntryWire {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
  readonly isNote: boolean
  readonly type: string
}

interface MoveResultWire {
  readonly from: string
  readonly to: string
  readonly repaired: readonly string[]
}

/** One row of a listing, kept as the plain value the window carries it as. */
export const mapEntry = (one: EntryWire): Entry => ({
  path: one.path,
  name: one.name,
  isFolder: one.isFolder,
  isNote: one.isNote,
  type: mapNoteType(one.type),
})

/** What the file did, in the shape the window carries it. */
export const mapMoveResult = (answer: MoveResultWire): MoveResult => ({
  from: answer.from,
  to: answer.to,
  repaired: answer.repaired,
})
