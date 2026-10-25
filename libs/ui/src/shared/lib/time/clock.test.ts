/** When work is put on the screen. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { onNextFrame } from './clock'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('work for the next frame', () => {
  it('waits for the frame where there is one', () => {
    const frames: ((now: number) => void)[] = []
    vi.stubGlobal('requestAnimationFrame', (run: (now: number) => void): number => frames.push(run))
    const run = vi.fn<() => void>()

    onNextFrame(run)

    expect(run).not.toHaveBeenCalled()
    frames[0]?.(0)
    expect(run).toHaveBeenCalledTimes(1)
  })

  it('runs now where there is no frame', () => {
    vi.stubGlobal('requestAnimationFrame', undefined)
    const run = vi.fn<() => void>()

    onNextFrame(run)

    expect(run).toHaveBeenCalledTimes(1)
  })
})
