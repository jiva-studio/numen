/**
 * What a branch does with the shares a splitter reports.
 *
 * A splitter moves the panels itself while a handle is dragged, and the model
 * is notified once, at the drop position.
 */
import { onBeforeUnmount } from 'vue'

/** Attribute that marks a dragged handle. It is set on the branch, which contains the panels the drag divides. */
export const RESIZING = 'data-resizing'

/** How close two shares are before they are the same share. */
const EPSILON = 1e-6

/** Whether two lists of shares stand at the same places. */
export const isSameShares = (a: readonly number[], b: readonly number[]): boolean =>
  a.length === b.length && a.every((share, index) => Math.abs(share - (b[index] ?? 0)) < EPSILON)

export interface BranchResizeOptions {
  readonly getBranchId: () => string
  readonly getShares: () => readonly number[]
  readonly getFrame: () => HTMLElement | undefined
  readonly resize: (branch: string, shares: readonly number[]) => void
}

export interface BranchResize {
  readonly onLayout: (percents: readonly number[]) => void
  readonly onHold: (isIndexed: boolean) => void
}

export function useBranchResize(options: BranchResizeOptions): BranchResize {
  let isDragging = false
  let pendingShares: readonly number[] | null = null

  /** The handle put down, wherever the pointer had reached by then. */
  const putDown = (): void => {
    if (!isDragging) return
    isDragging = false
    options.getFrame()?.removeAttribute(RESIZING)

    const resolved = pendingShares
    pendingShares = null
    if (resolved) options.resize(options.getBranchId(), resolved)
  }

  /**
   * A splitter reports the shares it settled on, including the ones it worked
   * out for itself when a panel arrived or left. Only a difference is passed on.
   */
  const onLayout = (percents: readonly number[]): void => {
    const shares = percents.map((size) => size / 100)
    if (isSameShares(shares, options.getShares())) return
    if (isDragging) pendingShares = shares
    else options.resize(options.getBranchId(), shares)
  }

  const onHold = (isIndexed: boolean): void => {
    if (!isIndexed) {
      putDown()
      return
    }
    isDragging = true
    options.getFrame()?.setAttribute(RESIZING, '')
  }

  // The branch is going, and where its handle reached is not news about a node
  // that will not be there to receive it.
  onBeforeUnmount(() => {
    pendingShares = null
    putDown()
  })

  return { onLayout, onHold }
}
