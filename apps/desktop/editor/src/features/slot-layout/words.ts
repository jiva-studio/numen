/**
 * The text of the menu that moves an entry of a bar. The slot titles are the names the product
 * gives the places an entry can be moved to.
 */
// Stryker disable StringLiteral: interface copy
import type { SlotName } from '@numen/plugin-host'

export const MOVE_WORDS = {
  moveTo: 'Move to',
  atTheEnd: 'At the end',
}

const SLOT_TITLES: Readonly<Partial<Record<SlotName, string>>> = {
  'titleBar.commands': 'Title bar buttons',
  'titleBar.status': 'Title bar status',
  'statusBar.left': 'Status bar start',
  'statusBar.right': 'Status bar end',
}

/** The title of a slot in the menu; a slot with no title is named by its slot name. */
export const getSlotTitle = (slot: SlotName): string => SLOT_TITLES[slot] ?? slot
