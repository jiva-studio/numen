import type { Position, Size } from '@/shared/lib/geometry'
import type { MenuItem, MenuTrigger } from './item'

export interface MenuProps {
  /** What can be chosen, in the order it is drawn. */
  items: readonly MenuItem[]
  /** Where it was asked for, in the coordinates of the area it is drawn into. */
  at: Position
  /** Whether it is drawn at all. */
  isOpen?: boolean
  /** What opened it. Opened by hand it appears with nothing chosen. */
  trigger?: MenuTrigger
  /**
   * Which item is the one in force, by the caller's identifier. A menu
   * naming one offers a choice between its items: each says whether it is
   * the one, and the keyboard opens on it.
   */
  current?: string | null
  /** Where the keyboard goes back to once it closes. */
  from?: HTMLElement | SVGElement | null
  /** The area it is placed in. The browser's own by default. */
  viewport?: Size | null
  /**
   * Whether the name of a group is drawn over it. A menu whose groups are
   * named by identifiers draws none.
   */
  hasGroups?: boolean
  /**
   * How wide what asked for it is. The menu is never narrower than that, and
   * grows past it for what it holds.
   */
  asking?: number
  /** Kept clear of that area's edges. */
  margin?: number
  /** Where it is drawn. The end of the document by default. */
  to?: string | HTMLElement
  /** What the moment is. The window's clock by default. */
  clock?: () => number
  /** What it is announced as. */
  name?: string
  /**
   * Whether a line to search the items by stands above them. The items shown
   * are those that name every word typed in their text, detail or group.
   */
  isSearchable?: boolean
  /** What the search line is announced as. */
  searchName?: string
}

export interface MenuEvents {
  /** An item was chosen. The identifier is the caller's, handed back as given. */
  (event: 'choose', id: string): void
  /** It asks to be put away. */
  (event: 'dismiss'): void
}

export interface MenuSlots {
  /**
   * What is drawn before an item's words. The room for it is kept on every
   * item once the slot is filled, so the words line up down the menu whether
   * or not each of them draws anything.
   */
  icon(props: { id: string }): unknown
  /** What is said when there is nothing to choose. */
  silence(): unknown
}
