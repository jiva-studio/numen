/**
 * The rows of a list that are drawn. A list that checks several draws only the rows in sight, by the height
 * of one row; until that height is measured it draws a first screen of them.
 */
import { computed, onBeforeUnmount, onMounted, shallowRef, type ComputedRef, type Ref } from 'vue'
import { measureElementRect } from '@/shared/api/dom'
import { useVirtualTree } from '@/shared/model/virtual'
import type { ListRow } from './rows'

const UNMEASURED_ROWS = 20

export interface RowWindowOptions {
  readonly list: Readonly<Ref<HTMLElement | null>>
  readonly rows: () => readonly ListRow[]
  readonly isMultiple: () => boolean
}

export interface RowWindowState {
  readonly first: ComputedRef<number>
  readonly shown: ComputedRef<readonly ListRow[]>
  readonly windowStyle: ComputedRef<Record<string, string> | undefined>
  readonly onScroll: () => void
  readonly scrollToRow: (index: number) => void
}

export function useRowWindow(options: RowWindowOptions): RowWindowState {
  const rowHeight = shallowRef(0)
  const virtual = useVirtualTree(
    options.list,
    () => options.rows().length,
    () => rowHeight.value,
  )

  const isWindowed = computed(() => options.isMultiple() && rowHeight.value > 0)

  const first = computed(() => (isWindowed.value ? virtual.window.value.startIndex : 0))

  const shown = computed(() => {
    if (!options.isMultiple()) return options.rows()
    if (!isWindowed.value) return options.rows().slice(0, UNMEASURED_ROWS)
    return options.rows().slice(virtual.window.value.startIndex, virtual.window.value.endIndex)
  })

  const windowStyle = computed(() =>
    isWindowed.value
      ? {
          blockSize: `${virtual.window.value.totalHeight}px`,
          paddingBlockStart: `${virtual.window.value.offsetY}px`,
          boxSizing: 'border-box',
        }
      : undefined,
  )

  let observer: ResizeObserver | null = null

  const measure = (): void => {
    const row = options.list.value?.querySelector('[role="option"]')
    if (row) rowHeight.value = measureElementRect(row).height
    virtual.measureViewport()
  }

  onMounted(() => {
    if (!options.isMultiple()) return
    measure()
    if (typeof ResizeObserver !== 'undefined' && options.list.value) {
      observer = new ResizeObserver(measure)
      observer.observe(options.list.value)
    }
  })

  onBeforeUnmount(() => {
    observer?.disconnect()
  })

  return { first, shown, windowStyle, onScroll: virtual.onScroll, scrollToRow: virtual.scrollToRow }
}
