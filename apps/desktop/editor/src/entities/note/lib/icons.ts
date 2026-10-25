/**
 * What each type of note is drawn as, wherever a note's type is drawn: the files list, the plex, and
 * the tab it opens in. A type a plugin declares is drawn with the icon of the tab kind that edits it,
 * and every other note as a note.
 */
import { shallowReactive, type Component } from 'vue'
import { FileText } from '@lucide/vue'
import type { NoteType } from '@/entities/file/@x/note'

const icons = shallowReactive(new Map<NoteType, Component>())

/** Draws the notes of a type with an icon, until the returned function is called. */
export function registerNoteIcon(type: NoteType, icon: Component): () => void {
  icons.set(type, icon)
  return () => {
    if (icons.get(type) === icon) icons.delete(type)
  }
}

/** The icon for a type of note. Every type has one. */
export const getNoteIcon = (type: NoteType): Component => icons.get(type) ?? FileText
