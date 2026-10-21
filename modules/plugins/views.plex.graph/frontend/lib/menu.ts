/**
 * The menu on a node, and the menu off every node: what each offers, and what
 * choosing an item evaluates to.
 */
import type { MenuItem } from '@numen/ui'
import { WORDS as words } from '../words'

/** The one the tab does itself: a note made where the picture relies on none. */
export const NEW_NOTE = 'newNote'

/** The groups the items stand in, in the order they are drawn. */
const BAND = {
  open: 'open',
  file: 'file',
  plex: 'plex',
  agent: 'agent',
  remove: 'remove',
} as const

/** What is offered on a node, in the order it is drawn. */
export const ITEMS: readonly MenuItem[] = [
  { id: 'read', text: words.read, group: BAND.open },
  { id: 'travel', text: words.travel, group: BAND.open },
  { id: 'preset', text: words.preset, group: BAND.open },
  { id: 'copy', text: words.copy, group: BAND.file },
  { id: 'reveal', text: words.reveal, group: BAND.file },
  { id: 'child', text: words.child, group: BAND.plex },
  { id: 'parent', text: words.parent, group: BAND.plex },
  { id: 'jump', text: words.jump, group: BAND.plex },
  { id: 'title', text: words.title, group: BAND.plex },
  { id: 'ask', text: words.ask, group: BAND.agent },
  { id: 'remove', text: words.remove, group: BAND.remove },
]

/** What is offered off every node. */
export const NONE: readonly MenuItem[] = [{ id: NEW_NOTE, text: words.newNote }]

/** What the menu offers, by identity. A choice outside this is not the menu's. */
export const OFFERED: ReadonlySet<string> = new Set(ITEMS.map((item) => item.id))
