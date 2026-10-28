/**
 * Browser DOM and clock platform ports.
 */
export const win: Window & typeof globalThis =
  typeof window !== 'undefined' ? window : ({} as Window & typeof globalThis)

export const doc: Document =
  typeof document !== 'undefined' ? document : ({} as Document)

export const nav: Navigator =
  typeof navigator !== 'undefined' ? navigator : ({} as Navigator)

export const hasWindow = (): boolean => typeof window !== 'undefined'
export const hasDocument = (): boolean => typeof document !== 'undefined'

export function measureRect(el: Element): DOMRect
export function measureRect(el: Element | null | undefined): DOMRect | undefined
export function measureRect(el: Element | null | undefined): DOMRect | undefined {
  return el ? el.getBoundingClientRect() : undefined
}

export const getNow = (): number => Date.now()
export const getCurrentDate = (): Date => new Date()
