/**
 * The menu on a node, and the menu off every node: what each offers, and what
 * choosing an item evaluates to.
 */
import type { MenuItem } from '@numen/ui'
import { WORDS as words } from '../words'

/** The command the tab handles itself: creates a note when the graph depends on none. */
export const NEW_NOTE = 'newNote'

/** The groups the items stand in, in the order they are drawn. */
const BAND = {
  open: 'open',
  file: 'file',
  plex: 'plex',
  declared: 'declared',
  remove: 'remove',
} as const

/** What the window does to a node's note, offered on every node in the order it is drawn. */
export const ITEMS: readonly MenuItem[] = [
  { id: 'read', text: words.read, group: BAND.open },
  { id: 'copy', text: words.copy, group: BAND.file },
  { id: 'child', text: words.child, group: BAND.plex },
  { id: 'parent', text: words.parent, group: BAND.plex },
  { id: 'jump', text: words.jump, group: BAND.plex },
  { id: 'title', text: words.title, group: BAND.plex },
  { id: 'remove', text: words.remove, group: BAND.remove },
]

/**
 * What is offered on a node: what opens its note, the commands the plugins
 * declare over a note in a band of their own, and the rest of what the window does.
 */
export const createNodeItems = (
  declared: readonly { readonly id: string; readonly text: string }[],
): readonly MenuItem[] => [
  ...ITEMS.filter((item) => item.group === BAND.open),
  ...declared.map((one) => ({ id: one.id, text: one.text, group: BAND.declared })),
  ...ITEMS.filter((item) => item.group !== BAND.open),
]

/** What is offered off every node. */
export const NONE: readonly MenuItem[] = [{ id: NEW_NOTE, text: words.newNote }]

/** What the menu offers, by identity. A choice outside this is not the menu's. */
export const OFFERED: ReadonlySet<string> = new Set(ITEMS.map((item) => item.id))
