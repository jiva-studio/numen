/**
 * The menu on a node: what it offers, and what choosing an item comes to.
 *
 * What it offers is every command over a note, so a person who never opens the
 * palette reaches all of them, and the two lists are one list. Apart from the
 * template, so that both can be asked without a screen.
 */
import type { MenuItem } from '@numen/ui'
import { commandsOf, overNote } from '../commanding'
import { WORDS as words } from '../words'

/** What is offered, in the order it is drawn. */
export const ITEMS: readonly MenuItem[] = overNote(commandsOf(words)).map(({ id, text }) => ({
  id,
  text,
}))

/** Where a choice goes. A command is carried out by the window, not by a menu. */
export interface Choices {
  /** A command asked for on a note, under the name the picture gives it. */
  runs(id: string, path: string, title: string): void
}

const offered = new Set(ITEMS.map((item) => item.id))

/** Carry out a choice. An identifier the menu does not offer does nothing. */
export function chose(id: string, path: string, title: string, choices: Choices): void {
  if (!offered.has(id)) return
  choices.runs(id, path, title)
}
