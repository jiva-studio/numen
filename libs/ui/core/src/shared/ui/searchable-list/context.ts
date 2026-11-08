/**
 * What a searchable list hands down to its rows without passing it through every
 * layer between: the slots the caller filled, and the query as written.
 */
import type { InjectionKey } from 'vue'
import type { SearchableListSlots } from './SearchableList.types'

export interface ListContext {
  readonly slots: SearchableListSlots
  readonly getQuery: () => string
}

export const LIST_CONTEXT: InjectionKey<ListContext> = Symbol('searchable-list')
