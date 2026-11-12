/**
 * Reactive tracking of visible window and viewport scroll for virtualized trees.
 */
import { computed, shallowRef, type ComputedRef, type Ref } from 'vue'
import { computeScrollTarget, computeVirtualWindow, type VirtualWindow } from '../lib/virtual'

export interface VirtualTreeState {
  readonly window: ComputedRef<VirtualWindow>
  readonly onScroll: () => void
  readonly measureViewport: () => void
  readonly scrollToRow: (index: number) => void
}

export function useVirtualTree(
  container: Ref<HTMLElement | null>,
  getTotalCount: () => number,
  getRowHeight: () => number,
): VirtualTreeState {
  const scrollTop = shallowRef(0)
  const viewportHeight = shallowRef(0)

  const window = computed<VirtualWindow>(() =>
    computeVirtualWindow({
      totalCount: getTotalCount(),
      rowHeight: getRowHeight(),
      viewportHeight: viewportHeight.value,
      scrollTop: scrollTop.value,
    }),
  )

  const onScroll = (): void => {
    if (container.value) {
      scrollTop.value = container.value.scrollTop
    }
  }

  const measureViewport = (): void => {
    if (container.value) {
      viewportHeight.value = container.value.clientHeight
      scrollTop.value = container.value.scrollTop
    }
  }

  const scrollToRow = (index: number): void => {
    if (!container.value) return
    const rowHeight = getRowHeight()
    const target = computeScrollTarget({
      targetIndex: index,
      rowHeight,
      viewportHeight: viewportHeight.value,
      currentScrollTop: container.value.scrollTop,
    })
    container.value.scrollTop = target
    scrollTop.value = target
  }

  return {
    window,
    onScroll,
    measureViewport,
    scrollToRow,
  }
}
