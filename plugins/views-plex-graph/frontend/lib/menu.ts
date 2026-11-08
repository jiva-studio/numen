/**
 * The menu off every node: what it offers. The menu on a node is the slot `plexGraph.context`.
 */
import type { MenuItem } from '@numen/ui'
import { WORDS as words } from '../words'

/** The command the tab handles itself: creates a note when the graph depends on none. */
export const NEW_NOTE = 'newNote'

/** What is offered off every node. */
export const NONE: readonly MenuItem[] = [{ id: NEW_NOTE, text: words.newNote }]
