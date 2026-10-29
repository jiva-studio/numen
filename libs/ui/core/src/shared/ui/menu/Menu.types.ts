import type { Position, Size } from '@/shared/lib/math/geometry'
import type { MenuItem, MenuTrigger } from './items/item'

export interface MenuProps {
  /** What can be chosen, in the order it is drawn. */
  items: readonly MenuItem[]
  /** The position of the request, in the coordinates of the area it is drawn into. */
  at: Position
  /** Whether it is drawn at all. */
  isOpen?: boolean
  /** What opened the menu. When opened by pointer it appears with no item chosen. */
  trigger?: MenuTrigger
  /**
   * The current item, by the caller's identifier. A menu with a current item
   * is a choice between its items: each item reports whether it is current, and
   * the keyboard opens on it.
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
   * The width of the element that requested the menu. The menu is at least that
   * wide and grows for its content.
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
   * Whether a search field is shown above the items. The items shown are those
   * that contain every typed word in their text, detail or group.
   */
  isSearchable?: boolean
  /** Whether a search query not matching any item is offered as a custom value. */
  canEnterCustom?: boolean
  /** What the search line is announced as. */
  searchName?: string
}

export interface MenuEvents {
  /** An item was chosen. The identifier is the caller's, handed back as given. */
  (event: 'choose', id: string): void
  /** The menu requests to be closed. */
  (event: 'dismiss'): void
}

export interface MenuSlots {
  /**
   * The content drawn before an item's text. Space for it is reserved on every
   * item once the slot is filled, so the text aligns down the menu whether or
   * not each item draws anything.
   */
  icon(props: { id: string }): unknown
  /** The content shown when there is nothing to choose. */
  silence(): unknown
}
