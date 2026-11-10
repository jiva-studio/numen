/**
 * The page helpers: the user agent and the window listeners.
 */
import { describe, expect, it, vi } from 'vitest'
import { getUserAgent, listenWindowPointerAndEscape } from './dom'

describe('dom helpers', () => {
  it('reads navigator.userAgent and registers/removes pointerdown and keydown listeners', () => {
    expect(getUserAgent()).toBe(navigator.userAgent)
    expect(getUserAgent().length).toBeGreaterThan(0)

    const onPointer = vi.fn()
    const onKey = vi.fn()
    const stop = listenWindowPointerAndEscape(onPointer, onKey)

    window.dispatchEvent(new Event('pointerdown'))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(onPointer).toHaveBeenCalledTimes(1)
    expect(onKey).toHaveBeenCalledTimes(1)

    stop()
    window.dispatchEvent(new Event('pointerdown'))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(onPointer).toHaveBeenCalledTimes(1)
    expect(onKey).toHaveBeenCalledTimes(1)
  })
})
