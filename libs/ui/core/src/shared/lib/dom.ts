/**
 * Browser DOM measurement and clock adapters.
 */

export function measureElementRect(element: Element): DOMRect {
  return element.getBoundingClientRect()
}

export function getNow(): number {
  return Date.now()
}
