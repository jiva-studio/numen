/**
 * What each note is drawn as, wherever a note is drawn: the files list, the palette and the tab it
 * opens in. A note is drawn with the icon of the tab kind the core chose to open it (FileKind.tab_kind),
 * and a note no tab kind with an icon opens as a note.
 */
import { shallowReactive, type Component } from 'vue'
import { FileText } from '@lucide/vue'

const icons = shallowReactive(new Map<string, Component>())

/** Draws the notes the tab kind opens with an icon, until the returned function is called. */
export function registerTabKindIcon(tabKind: string, icon: Component): () => void {
  icons.set(tabKind, icon)
  return () => {
    if (icons.get(tabKind) === icon) icons.delete(tabKind)
  }
}

/** The icon of a note by the tab kind that opens it; every note has one. */
export const getNoteIcon = (tabKind: string | undefined): Component =>
  (tabKind === undefined ? undefined : icons.get(tabKind)) ?? FileText
