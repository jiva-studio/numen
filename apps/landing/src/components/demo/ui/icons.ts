import type { Component } from 'vue'
import {
  AudioLines,
  Book,
  Bot,
  FileText,
  Folder,
  FolderOpen,
  FolderTree,
  Layers,
  Link,
  BookOpen,
  Waypoints,
} from '@lucide/vue'
import type { TabKind } from '../lib/tabs'

const TAB_ICONS: Readonly<Record<TabKind, Component>> = {
  files: FolderTree,
  plex: Waypoints,
  agent: Bot,
  note: FileText,
  pdf: BookOpen,
  recording: AudioLines,
  link: Link,
  deck: Layers,
}

export const getTabIcon = (kind: TabKind): Component => TAB_ICONS[kind]

/** The icon of a row of the file tree. */
export function getRowIcon(path: string, isOpen: boolean): Component {
  if (path.endsWith('.mp3')) return AudioLines
  if (path.endsWith('.pdf')) return Book
  if (path.endsWith('.md')) return path.startsWith('decks/') ? Layers : path.startsWith('lectures/') ? Link : FileText
  return isOpen ? FolderOpen : Folder
}
