import type { Viewport } from '@numen/ui'

/** A window that has reported its size at once, 1200 by 800, where a test has no layout engine to report one. */
export const fixedViewport: Viewport = {
  watch: (_of, took) => {
    took({ width: 1200, height: 800 })
    return () => undefined
  },
  watchFreeSpace: (took) => {
    took({ width: 1200, height: 800 })
    return () => undefined
  },
  watchWhole: () => () => undefined,
}
