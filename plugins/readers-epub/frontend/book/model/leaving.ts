/**
 * The copy of the spread in front that a turn into the adjacent document carries out, while the
 * paper already holds the markup of the next one.
 */

/** A copy of the spread in front, carried out by a turn into the adjacent document. */
export interface LeavingSpread {
  readonly copy: HTMLElement
  readonly target: number
  readonly remove: () => void
}

/**
 * Puts a copy of `paper` into `holder`, in place of any copy there, and returns it. `mark` gets
 * the copy while it still carries the run offsets, so it can read the runs. The copy then carries
 * no ids and no run offsets, so a lookup in the document finds only the paper.
 */
export function copyPaper(
  paper: HTMLElement,
  holder: HTMLElement,
  mark: (copy: HTMLElement) => void,
): HTMLElement {
  const copy = paper.cloneNode(true) as HTMLElement
  mark(copy)
  for (const element of copy.querySelectorAll('[id], [data-offset]')) {
    element.removeAttribute('id')
    element.removeAttribute('data-offset')
  }
  holder.replaceChildren(copy)
  return copy
}
