import type { Component } from 'vue'
import { File } from '@lucide/vue'
import { getNoteIcon } from '@/entities/note'
import { findIcon } from '@/shared/icons'
import { CREATE_NOTE_ID } from '../lib/offers/offers'
import type { Commands } from './palette/palette'
import type { SearchState } from './search/search'

export function findRowIcon(
  id: string,
  commands: Commands,
  search: SearchState,
  findFileIcon?: (path: string) => Component | undefined,
) {
  if (commands.open.value) {
    const picked = commands.findNoteType(id)
    return picked ? getNoteIcon(picked) : (commands.findCommandIcon(id) ?? findIcon(id))
  }
  if (id === CREATE_NOTE_ID) return findIcon('note')
  const type = search.findNoteType(id)
  if (type) return getNoteIcon(type)
  const file = search.findHitFile(id)
  return file ? (findFileIcon?.(file) ?? File) : null
}
