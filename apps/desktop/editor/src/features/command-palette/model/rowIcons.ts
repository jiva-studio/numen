import type { Component } from 'vue'
import { File } from '@lucide/vue'
import { getNoteIcon } from '@/entities/note'
import { findIcon } from '@/shared/icons'
import { COMMAND } from '@/shared/commandIds'
import { CREATE_NOTE_ROW } from '../lib/offers/offers'
import type { Commands } from './palette/palette'
import type { SearchState } from './search/search'

/** What the window knows of the files a search turns up. */
export interface FileIcons {
  /** The icon of the reader that opens a non-note file, or undefined when none does. */
  findFileIcon?(path: string): Component | undefined
}

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
  if (id === CREATE_NOTE_ROW) return findIcon(COMMAND.createNote)
  const type = search.findNoteType(id)
  if (type) return getNoteIcon(type)
  const file = search.findHitFile(id)
  return file ? (findFileIcon?.(file) ?? File) : null
}
