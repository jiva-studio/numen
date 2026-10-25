/**
 * Search filtering and keyboard interaction within a menu.
 */
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { filterItems, type MenuItem } from './item'
import { isLetter } from './typeahead'

export interface MenuSearchOptions {
  readonly isSearchable: () => boolean
  readonly items: () => readonly MenuItem[]
  readonly searchRef: Ref<HTMLInputElement | null>
  readonly menuRef: Ref<HTMLElement | null>
  readonly onRowKey: (event: KeyboardEvent) => void
  readonly onChoose: (item: MenuItem) => void
  readonly onResetHere: () => void
  readonly goToInitial: () => void
}

export interface MenuSearchState {
  readonly query: Ref<string>
  readonly visibleItems: ComputedRef<readonly MenuItem[]>
  readonly onKey: (event: KeyboardEvent) => void
  readonly focusInitial: () => void
}

export function useMenuSearch(options: MenuSearchOptions): MenuSearchState {
  const query = ref('')

  const visibleItems = computed(() =>
    options.isSearchable() ? filterItems(options.items(), query.value) : options.items(),
  )

  watch(query, () => options.onResetHere())

  const onKey = (event: KeyboardEvent): void => {
    if (!options.isSearchable()) return options.onRowKey(event)
    if (event.target === options.searchRef.value) {
      if (event.key === 'Enter') {
        event.preventDefault()
        const first = visibleItems.value.find((one) => !one.isDisabled)
        if (first && query.value) options.onChoose(first)
      } else if (['ArrowDown', 'ArrowUp', 'Tab'].includes(event.key)) {
        options.onResetHere()
        options.onRowKey(event)
      }
      return
    }
    if (isLetter(event)) options.searchRef.value?.focus()
    else options.onRowKey(event)
  }

  const focusInitial = (): void => {
    if (!options.isSearchable()) {
      options.goToInitial()
      return
    }
    options.searchRef.value?.focus()
    options.menuRef.value
      ?.querySelector('[aria-checked="true"]')
      ?.scrollIntoView?.({ block: 'nearest' })
  }

  return {
    query,
    visibleItems,
    onKey,
    focusInitial,
  }
}
