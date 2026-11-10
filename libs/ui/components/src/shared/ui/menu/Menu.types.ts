import type { VNode } from 'vue'
import type { Position, Size } from '@/shared/lib/math/geometry'
import type { MenuItem, MenuTrigger } from './items/item'

export interface MenuProps {
  items: readonly MenuItem[]
  at: Position
  isOpen?: boolean
  trigger?: MenuTrigger
  current?: string | null
  from?: HTMLElement | SVGElement | null
  viewport?: Size | null
  hasGroups?: boolean
  asking?: number
  margin?: number
  to?: string | HTMLElement
  clock?: () => number
  name?: string
}

export interface MenuEvents {
  /** An item was chosen. The identifier is the caller's, returned as given. */
  (event: 'choose', id: string): void
  /** The menu requests to be closed. */
  (event: 'dismiss'): void
  /**
   * The id of the item under the pointer, in this menu or a submenu, or null
   * when the pointer leaves the menu. Emitted once per change.
   */
  (event: 'highlight', id: string | null): void
}

export interface MenuSlots {
  /**
   * The content drawn before an item's text. Space for it is reserved on every
   * item once the slot is filled, so the text aligns down the menu whether or
   * not each item draws anything.
   */
  icon(props: { id: string }): VNode[]
  /** The content shown when there is nothing to choose. */
  silence(): VNode[]
}
