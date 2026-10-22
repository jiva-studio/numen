import { findSourceIcon } from '@/entities/file'
import { getNoteIcon } from '@/entities/note'
import { findIcon } from '@/shared/icons'
import { CREATE_NOTE_ID } from './offers'
import type { Commands } from '../model/palette'
import type { SearchState } from '../model/search'

export function findRowIcon(id: string, commands: Commands, search: SearchState) {
  if (commands.open.value) {
    const picked = commands.findNoteType(id)
    return picked ? getNoteIcon(picked) : findIcon(id)
  }
  if (id === CREATE_NOTE_ID) return findIcon('note')
  const type = search.findNoteType(id)
  if (type) return getNoteIcon(type)
  const kind = search.findHitSource(id)
  return kind ? findSourceIcon(kind) : null
}
