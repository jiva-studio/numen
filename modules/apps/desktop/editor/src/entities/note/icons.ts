/**
 * What each kind of note is drawn as, wherever a note's kind is drawn: the
 * files list, the plex, and the tab it opens in. One mark to a kind, so a
 * preset is the same thing in the tree that it is in the tab.
 */
import { FileText, Gauge, Layers, LayoutTemplate, type LucideIcon } from '@lucide/vue'
import type { NoteType } from './note'

const NOTES: ReadonlyMap<NoteType, LucideIcon> = new Map([
  ['note', FileText],
  ['deck', Layers],
  ['stencil', LayoutTemplate],
  ['preset', Gauge],
])

/** The icon for a kind of note. Every kind has one. */
export const iconOfNote = (type: NoteType): LucideIcon => NOTES.get(type) ?? FileText
