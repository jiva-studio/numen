/**
 * Makes everything outside the element inert: at each level from the element up to the body, every sibling that is
 * not inert already. The returned function gives back only what this call took.
 */
export function makeOutsideInert(element: HTMLElement): () => void {
  const changed: HTMLElement[] = []
  let node: HTMLElement = element
  while (node.parentElement !== null) {
    for (const sibling of Array.from(node.parentElement.children)) {
      if (sibling === node || !(sibling instanceof HTMLElement) || sibling.inert) continue
      sibling.inert = true
      changed.push(sibling)
    }
    node = node.parentElement
  }
  return () => {
    for (const one of changed) one.inert = false
  }
}
