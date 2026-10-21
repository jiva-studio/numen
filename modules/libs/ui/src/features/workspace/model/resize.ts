/**
 * What a branch does with the shares a splitter reports.
 *
 * A splitter moves the panels itself while a handle is held, and the model is
 * told once, where the handle came to rest.
 */
import { onBeforeUnmount } from 'vue'

/** What a held handle is marked by, on the branch so that it reaches the panels the drag divides. */
export const RESIZING = 'data-resizing'

/** How close two shares are before they are the same share. */
const EPSILON = 1e-6

/** Whether two lists of shares stand at the same places. */
export const isSameShares = (a: readonly number[], b: readonly number[]): boolean =>
  a.length === b.length && a.every((share, index) => Math.abs(share - (b[index] ?? 0)) < EPSILON)

export interface BranchResizeOptions {
  /** The branch being resized. */
  readonly getBranchId: () => string
  /** The shares the model holds. */
  readonly getShares: () => readonly number[]
  /** The branch's own element, which carries the mark. */
  readonly getFrame: () => HTMLElement | undefined
  /** The model told where the shares came to rest. */
  readonly resize: (branch: string, shares: readonly number[]) => void
}

export interface BranchResize {
  /** The shares a splitter settled on, in percent. */
  readonly onLayout: (percents: readonly number[]) => void
  /** A handle taken up, and put down where it stopped. */
  readonly onHold: (isHeld: boolean) => void
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

  const onHold = (isHeld: boolean): void => {
    if (!isHeld) {
      putDown()
      return
    }
    isDragging = true
    options.getFrame()?.setAttribute(RESIZING, '')
  }

  // The branch is going, and where its handle reached is not news about a node
  // that will not be there to hear it.
  onBeforeUnmount(() => {
    pendingShares = null
    putDown()
  })

  return { onLayout, onHold }
}
