import type { Position, Size } from '@/shared/lib/math/geometry'
import type { ListItem } from './rows'

export interface SearchableListProps {
  items: readonly ListItem[]
  label: string
  placeholder?: string
  emptyText?: string
  pendingText?: string
  maxRows?: number
  isFiltered?: boolean
  isPending?: boolean
  hasAction?: boolean
  needsQuery?: boolean
}

export interface SearchableListPopupProps {
  at: Position
  asking?: number
  viewport?: Size | null
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
