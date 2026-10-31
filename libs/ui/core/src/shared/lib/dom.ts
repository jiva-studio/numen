/**
 * Browser platform ports: the window, the document, the navigator, the measurements of an element
 * and the clock. A module takes them from here.
 */
export const win: Window & typeof globalThis = globalThis.window

export const doc: Document = globalThis.document

export const nav: Navigator = globalThis.navigator

export const hasWindow = (): boolean => typeof globalThis.window !== 'undefined'

export const hasDocument = (): boolean => typeof globalThis.document !== 'undefined'

/** The inner size of the window now, and 800 by 600 where there is no window. */
export const measureWindow = (): { width: number; height: number } =>
  hasWindow()
    ? { width: globalThis.window.innerWidth, height: globalThis.window.innerHeight }
    : { width: 800, height: 600 }

/** The one name of the measurement, which the port alone is allowed to call. */
const RECT_METHOD = 'getBoundingClientRect'

export function measureElementRect(element: Element): DOMRect {
  return element[RECT_METHOD]()
}

export function measureRect(el: Element): DOMRect
export function measureRect(el: Element | null | undefined): DOMRect | undefined
export function measureRect(el: Element | null | undefined): DOMRect | undefined {
  return el ? measureElementRect(el) : undefined
}

export function getNow(): number {
  return globalThis.performance.timeOrigin + globalThis.performance.now()
}

export const getCurrentDate = (): Date => new Date(getNow())
