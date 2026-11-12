/**
 * Virtual scroll and dimensions layout for tree rows.
 */
import { computed, onBeforeUnmount, onMounted, type ComputedRef, type Ref } from 'vue'
import type { ShownRow } from '../lib/row'
import { useVirtualTree, type VirtualTreeState } from './virtual'

export interface TreeLayoutOptions {
  readonly box: Ref<HTMLElement | null>
  readonly list: Ref<HTMLElement | null>
  readonly visibleRows: ComputedRef<readonly ShownRow[]>
  readonly getRowHeight: () => number
}

export interface TreeLayoutState {
  readonly virtual: VirtualTreeState
  readonly virtualRows: ComputedRef<readonly ShownRow[]>
  readonly rowsStyle: ComputedRef<{
    readonly blockSize: string
    readonly paddingBlockStart: string
    readonly boxSizing: 'border-box'
  }>
}

export function useTreeLayout(options: TreeLayoutOptions): TreeLayoutState {
  const virtual = useVirtualTree(
    options.box,
    () => options.visibleRows.value.length,
    options.getRowHeight,
  )

  const virtualRows = computed(() =>
    options.visibleRows.value.slice(virtual.window.value.startIndex, virtual.window.value.endIndex),
  )

  const rowsStyle = computed(() => ({
    blockSize: `${virtual.window.value.totalHeight}px`,
    paddingBlockStart: `${virtual.window.value.offsetY}px`,
    boxSizing: 'border-box' as const,
  }))

  let observer: ResizeObserver | null = null

  onMounted(() => {
    virtual.measureViewport()
    if (typeof ResizeObserver !== 'undefined' && options.box.value) {
      observer = new ResizeObserver(() => {
        virtual.measureViewport()
      })
      observer.observe(options.box.value)
    }
  })

  onBeforeUnmount(() => {
    observer?.disconnect()
  })

  return {
    virtual,
    virtualRows,
    rowsStyle,
  }
}
