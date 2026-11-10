import type { Component } from 'vue'
import {
  ArrowRightLeft,
  AudioLines,
  Book,
  BookOpen,
  Bot,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FilePlus,
  FileText,
  Folder,
  FolderOpen,
  FolderPlus,
  FolderTree,
  Layers,
  Link,
  PenLine,
  Plus,
  Trash2,
  Type,
  Waypoints,
} from '@lucide/vue'
import { DEMO_COMMAND } from '../lib/commands'
import type { Entry } from '../lib/entries'
import { DECK_PATH, YOUTUBE_LINK_PATH, getKindOfTab, type TabKind } from '../lib/tabs'
import { NEW_SUBMENU, RELATION_SUBMENU } from '../lib/menus'

const COMMAND_ICONS: Readonly<Record<string, Component>> = {
  [DEMO_COMMAND.openNote]: FileText,
  [DEMO_COMMAND.openNoteBeside]: FileText,
  [DEMO_COMMAND.revealNote]: FolderOpen,
  [DEMO_COMMAND.showNoteInPlex]: Waypoints,
  [DEMO_COMMAND.createChildNote]: CornerDownRight,
  [DEMO_COMMAND.createParentNote]: CornerLeftUp,
  [DEMO_COMMAND.createJumpNote]: ArrowRightLeft,
  [DEMO_COMMAND.renameNote]: Type,
  [DEMO_COMMAND.copyNotePath]: Copy,
  [DEMO_COMMAND.removeFile]: Trash2,
  [DEMO_COMMAND.createNote]: FilePlus,
  [DEMO_COMMAND.openPlex]: Waypoints,
  [DEMO_COMMAND.openFiles]: FolderTree,
  [DEMO_COMMAND.openAgent]: Bot,
  [DEMO_COMMAND.askAgent]: Bot,
  [DEMO_COMMAND.createFolder]: FolderPlus,
  [DEMO_COMMAND.renameTreeEntry]: PenLine,
  [NEW_SUBMENU]: Plus,
  [RELATION_SUBMENU]: Waypoints,
}

/** The icon a command is drawn with in a menu or the palette. */
export const getCommandIcon = (id: string): Component | null => COMMAND_ICONS[id] ?? null

const TAB_ICONS: Readonly<Record<TabKind, Component>> = {
  files: FolderTree,
  plex: Waypoints,
  agent: Bot,
  note: FileText,
  'reader:pdf': BookOpen,
  recording: AudioLines,
  link: Link,
  deck: Layers,
}

export function getTabIcon(id: string): Component {
  const kind = getKindOfTab(id)
  return kind ? TAB_ICONS[kind] : FileText
}

/** The icon of a row of the file tree. */
export function getEntryIcon(entry: Entry | undefined, isOpen: boolean): Component {
  if (!entry) return FileText
  if (entry.isFolder) return isOpen ? FolderOpen : Folder
  if (entry.type === 'deck') return Layers
  if (entry.type === 'link') return Link
  if (entry.path.endsWith('.mp3')) return AudioLines
  return entry.path.endsWith('.pdf') ? Book : FileText
}

/** The icon of a file in a list of results: by what it is. */
export function getFileIcon(id: string, isNote: boolean): Component | null {
  if (id.endsWith('.pdf')) return BookOpen
  if (id.endsWith('.mp3')) return AudioLines
  if (id === YOUTUBE_LINK_PATH) return Link
  if (id === DECK_PATH) return Layers
  return isNote ? FileText : null
}
