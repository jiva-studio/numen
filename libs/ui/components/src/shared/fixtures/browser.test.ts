import { afterEach, expect, it, vi } from 'vitest'
import { getVisibleFraction } from './browser'

afterEach(() => vi.unstubAllGlobals())

it.each([
  { width: 0, height: 10, ratio: 1, visible: 0 },
  { width: 10, height: 0, ratio: 1, visible: 0 },
  { width: 10, height: 10, ratio: 0.5, visible: 0.5 },
])('returns $visible for a $width by $height intersection at ratio $ratio', async (sample) => {
  const disconnect = vi.fn()
  vi.stubGlobal(
    'IntersectionObserver',
    class {
      constructor(readonly notify: (entries: Partial<IntersectionObserverEntry>[]) => void) {}
      disconnect = disconnect
      observe() {
        this.notify([
          {
            intersectionRect: new DOMRect(0, 0, sample.width, sample.height),
            intersectionRatio: sample.ratio,
          },
        ])
      }
    },
  )
  await expect(getVisibleFraction(document.createElement('div'))).resolves.toBe(sample.visible)
  expect(disconnect).toHaveBeenCalledOnce()
})
