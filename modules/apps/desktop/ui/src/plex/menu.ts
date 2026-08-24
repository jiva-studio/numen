/**
 * The menu on a node: what it offers, and what choosing an item comes to.
 *
 * What it offers is every command over a note that has a row of its own. A
 * command another one reaches on its row is left out, and the palette is where
 * it is reached. Apart from the template.
 */
import type { MenuItem } from '@numen/ui'
import { commandsOf, overNote } from '../commanding'
import { WORDS as words } from '../words'

/** What is offered, in the order it is drawn. */
export const ITEMS: readonly MenuItem[] = overNote(commandsOf(words)).map(({ id, text }) => ({
  id,
  text,
}))

/** What the menu offers, by identity. A choice outside this is not the menu's. */
export const OFFERED: ReadonlySet<string> = new Set(ITEMS.map((item) => item.id))
