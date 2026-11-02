/**
 * The rows of a searchable list and the one the keyboard is on.
 *
 * The active row is held by its key, so it stays where the user put it while
 * the items are replaced under it, and the first row keeps its key while an
 * action gives way to an item.
 */
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { buildRows, filterItems, stepRow, type ListItem, type ListRow } from './rows'

export interface ListRowsOptions {
  readonly items: () => readonly ListItem[]
  readonly query: Readonly<Ref<string>>
  /** The identifier of the item in force. */
  readonly current: () => string | null
  readonly isFiltered: () => boolean
  readonly hasAction: () => boolean
  readonly needsQuery: () => boolean
}

export interface ListRowsState {
  readonly rows: ComputedRef<readonly ListRow[]>
  /** The query without the space around it. */
  readonly queryText: ComputedRef<string>
  /** Whether the rows are drawn: always, or only once something is typed. */
  readonly isListOpen: ComputedRef<boolean>
  readonly activeKey: Ref<string | null>
  readonly activeIndex: ComputedRef<number>
  /** Moves the active row by a step, wrapping at the ends. */
  readonly moveBy: (step: number) => void
  /** Moves the active row to the first row or the last. */
  readonly moveToEnd: (isFirst: boolean) => void
  /** The row with the key. */
  readonly findRow: (key: string) => ListRow | undefined
}

export function useListRows(options: ListRowsOptions): ListRowsState {
  const activeKey = ref<string | null>(null)

  const queryText = computed(() => options.query.value.trim())

  const matches = computed(() =>
    options.isFiltered() ? filterItems(options.items(), options.query.value) : options.items(),
  )

  const rows = computed(() => buildRows(matches.value, options.query.value, options.hasAction()))

  const isListOpen = computed(() => !options.needsQuery() || queryText.value !== '')

  /** Where the keyboard starts: on the item in force while nothing is typed, else the first row. */
  const startIndex = computed(() => {
    if (queryText.value !== '') return 0
    return Math.max(
      rows.value.findIndex((row) => row.key === options.current()),
      0,
    )
  })

  const activeIndex = computed(() => {
    const found = rows.value.findIndex((row) => row.key === activeKey.value)
    return found < 0 ? startIndex.value : found
  })

  // A new query puts the keyboard back on the first row.
  watch(options.query, () => {
    activeKey.value = null
  })

  const moveBy = (step: number): void => {
    const next = rows.value[stepRow(rows.value.length, activeIndex.value, step)]
    if (next) activeKey.value = next.key
  }

  const moveToEnd = (isFirst: boolean): void => {
    activeKey.value = rows.value.at(isFirst ? 0 : -1)?.key ?? null
  }

  const findRow = (key: string): ListRow | undefined => rows.value.find((one) => one.key === key)

  return { rows, queryText, isListOpen, activeKey, activeIndex, moveBy, moveToEnd, findRow }
}
