/**
 * The copy of the spread in front that a turn into the adjacent document carries out, while the
 * paper already holds the markup of the next one.
 */

/**
 * Puts a copy of `paper` into `holder`, in place of any copy there, and returns it. The copy
 * carries no ids and no run offsets, so a lookup in the document finds only the paper.
 */
export function copyPaper(paper: HTMLElement, holder: HTMLElement): HTMLElement {
  const copy = paper.cloneNode(true) as HTMLElement
  for (const element of copy.querySelectorAll('[id], [data-offset]')) {
    element.removeAttribute('id')
    element.removeAttribute('data-offset')
  }
  holder.replaceChildren(copy)
  return copy
}
