/**
 * File domain conversions between schema representations and window models.
 */
import { mapNoteType } from '@/shared/api/wire'
import type { Entry } from '@/shared/model/fileTypes'

interface EntryWire {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
  readonly isNote: boolean
  readonly type: string
  readonly tabKind?: string
  readonly modifiedAtMs?: bigint | number
}

/** One row of a listing, kept as the plain value the window carries it as. */
export const mapEntry = (one: EntryWire): Entry => ({
  path: one.path,
  name: one.name,
  isFolder: one.isFolder,
  isNote: one.isNote,
  type: mapNoteType(one.type),
  ...(one.tabKind ? { tabKind: one.tabKind } : {}),
  ...(one.modifiedAtMs ? { modifiedAt: Number(one.modifiedAtMs) } : {}),
})
