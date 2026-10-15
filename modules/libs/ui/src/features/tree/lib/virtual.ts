/**
 * Visible window computation and scroll target calculations.
 */

export interface VirtualWindow {
  readonly startIndex: number
  readonly endIndex: number
  readonly offsetY: number
  readonly totalHeight: number
}

export interface VirtualConfig {
  readonly totalCount: number
  readonly rowHeight: number
  readonly viewportHeight: number
  readonly scrollTop: number
  readonly overscan?: number
}

export function computeVirtualWindow(config: VirtualConfig): VirtualWindow {
  const { totalCount, rowHeight, viewportHeight, scrollTop, overscan = 5 } = config
  const totalHeight = totalCount * rowHeight

  // An unmeasured container shows every row.
  if (viewportHeight <= 0 || rowHeight <= 0) {
    return { startIndex: 0, endIndex: totalCount, offsetY: 0, totalHeight }
  }

  const startNode = Math.floor(scrollTop / rowHeight)
  const visibleCount = Math.ceil(viewportHeight / rowHeight)

  const startIndex = Math.max(0, startNode - overscan)
  const endIndex = Math.min(totalCount, startNode + visibleCount + overscan)
  const offsetY = startIndex * rowHeight

  return { startIndex, endIndex, offsetY, totalHeight }
}

export interface ScrollTargetConfig {
  readonly targetIndex: number
  readonly rowHeight: number
  readonly viewportHeight: number
  readonly currentScrollTop: number
}

export function computeScrollTarget(config: ScrollTargetConfig): number {
  const { targetIndex, rowHeight, viewportHeight, currentScrollTop } = config
  if (viewportHeight <= 0 || rowHeight <= 0) {
    return currentScrollTop
  }

  const targetTop = targetIndex * rowHeight
  const targetBottom = targetTop + rowHeight

  if (targetTop < currentScrollTop) {
    return targetTop
  }
  if (targetBottom > currentScrollTop + viewportHeight) {
    return targetBottom - viewportHeight
  }
  return currentScrollTop
}
