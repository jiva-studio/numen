import type { Component } from 'vue'
import { File, Undo2 } from '@lucide/vue'
import { getNoteIcon } from '@/entities/note'
import { findIcon } from '@/shared/model/command'
import { KEEP_SUFFIX } from './view/stepGroups'
import type { Commands } from './palette/palette'
import type { SearchState } from './search/search'

/** What the window knows of the files a search turns up. */
export interface FileIcons {
  /** The icon of the reader that opens a non-note file, or undefined when none does. */
  findFileIcon?(this: void, path: string): Component | undefined
}

export function findRowIcon(
  id: string,
  commands: Commands,
  search: SearchState,
  findFileIcon?: (path: string) => Component | undefined,
) {
  if (id.endsWith(KEEP_SUFFIX)) return Undo2
  if (commands.palette.open.value) {
    const picked = commands.view.findTabKind(id)
    return picked === null
      ? (commands.view.findCommandIcon(id) ?? findIcon(id))
      : getNoteIcon(picked)
  }
  const tabKind = search.reads.findTabKind(id)
  if (tabKind !== null) return getNoteIcon(tabKind)
  const file = search.reads.findHitFile(id)
  return file ? (findFileIcon?.(file) ?? File) : null
}
