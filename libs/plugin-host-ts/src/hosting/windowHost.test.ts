/**
 * The window the plugins of a desktop window run in: the state of its frame, and the window buttons,
 * which go to the frame.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { WindowState } from '@numen/plugin-sdk'
import { createWindowHost, type WindowControls } from './windowHost'

function createFrame() {
  const isMaximized = ref(false)
  const isFullscreen = ref(false)
  const frame = {
    isMaximized,
    isFullscreen,
    minimizeWindow: vi.fn(async () => undefined),
    toggleMaximizeWindow: vi.fn(async () => undefined),
    closeWindow: vi.fn(async () => undefined),
  } satisfies WindowControls
  return { frame, isMaximized, isFullscreen }
}

describe('the window of the plugins', () => {
  it('reads the maximized and the full screen state of the frame', () => {
    const { frame, isFullscreen } = createFrame()
    isFullscreen.value = true

    expect(createWindowHost(frame, vi.fn()).getState()).toStrictEqual({
      isMaximized: false,
      isFullscreen: true,
    })
  })

  it('calls a watcher at once and on every change of the frame, until it is disposed', async () => {
    const { frame, isMaximized } = createFrame()
    const states: WindowState[] = []
    const watch = createWindowHost(frame, vi.fn()).watch((state) => states.push(state))

    isMaximized.value = true
    await nextTick()
    watch.dispose()
    isMaximized.value = false
    await nextTick()

    expect(states.map((one) => one.isMaximized)).toStrictEqual([false, true])
  })

  it('minimizes, toggles maximize and closes through the frame', () => {
    const { frame } = createFrame()
    const window = createWindowHost(frame, vi.fn())

    window.minimize()
    window.toggleMaximize()
    window.close()

    expect(frame.minimizeWindow).toHaveBeenCalledOnce()
    expect(frame.toggleMaximizeWindow).toHaveBeenCalledOnce()
    expect(frame.closeWindow).toHaveBeenCalledOnce()
  })

  it('reports a window button the frame fails to carry out', async () => {
    const { frame } = createFrame()
    const failure = new Error('The window was not closed')
    frame.closeWindow.mockRejectedValueOnce(failure)
    const onError = vi.fn()

    createWindowHost(frame, onError).close()
    await Promise.resolve()

    expect(onError).toHaveBeenCalledWith(failure)
  })
})
