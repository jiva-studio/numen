/**
 * The menu on a node: what it offers, and what choosing an item comes to.
 *
 * Apart from the template, so that both can be asked without a screen.
 */
import type { MenuItem } from '@numen/ui'

/** What is offered, in the order it is drawn. */
export const ITEMS: readonly MenuItem[] = [
  { id: 'open', text: 'Open in a tab' },
  { id: 'child', text: 'New child note' },
  { id: 'ask', text: 'Ask the agent about this note' },
  { id: 'copy', text: 'Copy path' },
]

/** What each item does to the note the menu was asked for on. */
export interface Choices {
  readonly open: (path: string) => void
  readonly child: (path: string) => void
  readonly ask: (path: string) => void
  readonly copy: (path: string) => void
}

const offered = new Set(ITEMS.map((item) => item.id))

/** Carry out a choice. An identifier the menu does not offer does nothing. */
export function chose(id: string, path: string, choices: Choices): void {
  if (!offered.has(id)) return
  choices[id as keyof Choices](path)
}
