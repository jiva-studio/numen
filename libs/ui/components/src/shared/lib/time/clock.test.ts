/** When work is shown. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { browserClock, getMonotonicNow, onNextFrame } from './clock'

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

describe('the browser clock', () => {
  it('reads the monotonic clock, which only goes forward', () => {
    const before = getMonotonicNow()
    expect(getMonotonicNow()).toBeGreaterThanOrEqual(before)
    expect(browserClock.now()).toBeGreaterThanOrEqual(before)
  })

  it('schedules a frame and takes it back by its handle', () => {
    const request = vi.fn<(run: (now: number) => void) => number>(() => 7)
    const cancel = vi.fn<(handle: number) => void>()
    vi.stubGlobal('requestAnimationFrame', request)
    vi.stubGlobal('cancelAnimationFrame', cancel)
    const run = vi.fn<(now: number) => void>()

    const handle = browserClock.schedule(run)
    browserClock.cancel(handle)

    expect(request).toHaveBeenCalledWith(run)
    expect(handle).toBe(7)
    expect(cancel).toHaveBeenCalledWith(7)
  })
})
