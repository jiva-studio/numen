import { measureElementRect, measureWindow } from '../api/dom'

/** The fraction of an element visible within its scrolling ancestor or the viewport. */
export function getVisibleFraction(element: Element, root: Element | null = null): Promise<number> {
  return new Promise((resolve) => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry) return
        observer.disconnect()
        resolve(entry.intersectionRatio)
      },
      { root },
    )
    observer.observe(element)
  })
}

/** Client coordinates for pointing at an element or a range of text. Fractions run from its start to its end. */
export function getPointerPosition(element: Element | Range, across = 0.5, down = 0.5) {
  const box = measureElementRect(element)
  return { clientX: box.left + box.width * across, clientY: box.top + box.height * down }
}

/** Whether two elements follow each other on screen without overlapping on the given axis. */
export function isBefore(
  first: Element,
  second: Element,
  axis: 'horizontal' | 'vertical',
): boolean {
  const a = measureElementRect(first)
  const b = measureElementRect(second)
  return axis === 'horizontal' ? a.right <= b.left : a.bottom <= b.top
}

/** Map a client point from the story iframe into the viewport where the browser drives its mouse. */
export function getBrowserPointerPosition(point: { x: number; y: number }) {
  const frame = window.frameElement
  if (!frame) return point
  const box = measureElementRect(frame)
  const viewport = measureWindow()
  return {
    x: box.left + point.x * (box.width / viewport.width),
    y: box.top + point.y * (box.height / viewport.height),
  }
}
