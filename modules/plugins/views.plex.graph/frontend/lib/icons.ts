/**
 * Icons used within the knowledge graph view.
 */
import {
  ArrowRightLeft,
  Bot,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FilePlus,
  FileText,
  FolderOpen,
  Gauge,
  Layers,
  LayoutTemplate,
  Trash2,
  Type,
  Waypoints,
  type LucideIcon,
} from '@lucide/vue'
import type { NoteType } from '../types'

const NOTE_ICONS: ReadonlyMap<NoteType, LucideIcon> = new Map([
  ['note', FileText],
  ['deck', Layers],
  ['stencil', LayoutTemplate],
  ['preset', Gauge],
])

export const getNoteIcon = (type: NoteType): LucideIcon => NOTE_ICONS.get(type) ?? FileText

const COMMAND_ICONS: ReadonlyMap<string, LucideIcon> = new Map([
  ['read', FileText],
  ['beside', FileText],
  ['travel', Waypoints],
  ['child', CornerDownRight],
  ['parent', CornerLeftUp],
  ['jump', ArrowRightLeft],
  ['title', Type],
  ['ask', Bot],
  ['copy', Copy],
  ['reveal', FolderOpen],
  ['preset', Gauge],
  ['remove', Trash2],
  ['newNote', FilePlus],
])

export const findIcon = (id: string): LucideIcon | null => COMMAND_ICONS.get(id) ?? null
