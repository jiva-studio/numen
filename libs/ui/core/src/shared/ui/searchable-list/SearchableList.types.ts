import type { Position, Size } from '@/shared/lib/math/geometry'
import type { ListItem } from './rows'

export interface SearchableListProps {
  /** What is offered, in the order it is drawn. */
  items: readonly ListItem[]
  /** What the list and its frame are announced as. */
  label: string
  /** What the search line says while empty, and what it is announced as. */
  placeholder?: string
  /** What is shown where no row is left. */
  emptyText?: string
  /** What is shown while the answer is awaited and no row is there yet. */
  pendingText?: string
  /** How many rows are visible before the list scrolls. */
  maxRows?: number
  /**
   * Whether the list narrows itself to the query. A caller whose items already
   * answer the query, as a search of its own does, turns it off.
   */
  isFiltered?: boolean
  /** Whether an answer is awaited. The rows stay where they are meanwhile. */
  isPending?: boolean
  /**
   * Whether an action row stands first, in the `action` slot, while a query is
   * typed. It stays row 1 as the items arrive; an item whose text is exactly the
   * query takes its place.
   */
  hasAction?: boolean
  /** Whether the rows are drawn only once something is typed. */
  needsQuery?: boolean
}

export interface SearchableListPopupProps {
  /** Where the popup is drawn, in the coordinates of the window. */
  at: Position
  /** The width of the element that asked for the popup, which is the least it measures. */
  asking?: number
  /** The area the popup is placed in. The browser's own by default. */
  viewport?: Size | null
  /** Where the keyboard goes back to once the popup closes. */
  from?: HTMLElement | null
}

export interface SearchableListSlots {
  /** What is drawn before the words of an item. */
  leading(props: { item: ListItem }): unknown
  /** What is drawn after the words of an item. */
  trailing(props: { item: ListItem }): unknown
  /** The action row that stands first, given the query as typed. */
  action(props: { query: string }): unknown
}
