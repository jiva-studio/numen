import { describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { MAC_BUTTONS_INSET } from '../lib/sizes'
import type { FrameRuntime } from '../lib/runtime'
import { getWindowEdge, useWindowFrame } from './useWindowFrame'

function createRuntime(platform: 'windows' | 'mac' | 'linux', maximized = false) {
  const state = { maximized }
  const calls = {
    Minimise: vi.fn(async () => undefined),
    ToggleMaximise: vi.fn(async () => {
      state.maximized = !state.maximized
    }),
    Close: vi.fn(async () => undefined),
  }
  const runtime: FrameRuntime = {
    Window: { ...calls, IsMaximised: async () => state.maximized },
    System: {
      IsWindows: () => platform === 'windows',
      IsMac: () => platform === 'mac',
      IsLinux: () => platform === 'linux',
    },
  }
  return { runtime, calls }
}

describe('useWindowFrame', () => {
  it('minimizes, toggles maximize and closes through the Wails window', async () => {
    const { runtime, calls } = createRuntime('windows')
    const frame = useWindowFrame(async () => runtime)

    await frame.minimizeWindow()
    await frame.toggleMaximizeWindow()
    await frame.closeWindow()

    expect(calls.Minimise).toHaveBeenCalledOnce()
    expect(calls.ToggleMaximise).toHaveBeenCalledOnce()
    expect(calls.Close).toHaveBeenCalledOnce()
  })

  it('reports the maximized state of the window, also after it is toggled', async () => {
    const { runtime } = createRuntime('windows', true)
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()
    expect(frame.isMaximized.value).toBe(true)

    await frame.toggleMaximizeWindow()

    expect(frame.isMaximized.value).toBe(false)
  })

  it('reports Windows as the platform on Windows', async () => {
    const frame = useWindowFrame(async () => createRuntime('windows').runtime)
    await flushPromises()

    expect(frame.platform.value).toBe('windows')
  })

  it('reports macOS as the platform on macOS', async () => {
    const frame = useWindowFrame(async () => createRuntime('mac').runtime)
    await flushPromises()

    expect(frame.platform.value).toBe('darwin')
  })

  it('reports Linux as the platform on Linux', async () => {
    const frame = useWindowFrame(async () => createRuntime('linux').runtime)
    await flushPromises()

    expect(frame.platform.value).toBe('linux')
  })

  it('does nothing and stays unmaximized where Wails does not serve the page', async () => {
    const frame = useWindowFrame(async () => undefined)
    await flushPromises()

    await expect(frame.minimizeWindow()).resolves.toBeUndefined()
    await expect(frame.toggleMaximizeWindow()).resolves.toBeUndefined()
    await expect(frame.closeWindow()).resolves.toBeUndefined()
    expect(frame.isMaximized.value).toBe(false)
    expect(frame.platform.value).toBe('windows')
  })
})

describe('useWindowFrame when the read of the maximized state fails', () => {
  /** A runtime whose state read resolves once, then rejects, as a Wails window that went away mid-call. */
  function createFailingRuntime(maximized: boolean) {
    const failing = { value: false }
    const runtime: FrameRuntime = {
      Window: {
        ToggleMaximise: async () => {
          failing.value = true
        },
        IsMaximised: async () => {
          if (failing.value) throw new Error('the window state is not available')
          return maximized
        },
      },
    }
    return { runtime, failing }
  }

  it('resolves the toggle although the read of the state after it rejects', async () => {
    const { runtime } = createFailingRuntime(true)
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()

    await expect(frame.toggleMaximizeWindow()).resolves.toBeUndefined()
  })

  it('keeps the state it had when the read after the toggle rejects', async () => {
    const { runtime } = createFailingRuntime(true)
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()

    await frame.toggleMaximizeWindow().catch(() => undefined)

    expect(frame.isMaximized.value).toBe(true)
  })

  it('stays unmaximized when the first read of the state rejects', async () => {
    const runtime: FrameRuntime = {
      Window: {
        IsMaximised: async () => {
          throw new Error('the window state is not available')
        },
      },
    }
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()

    expect(frame.isMaximized.value).toBe(false)
  })

  it('keeps the state it had when the read after a resize rejects', async () => {
    const { runtime, failing } = createFailingRuntime(true)
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()

    failing.value = true
    globalThis.dispatchEvent(new Event('resize'))
    await flushPromises()

    expect(frame.isMaximized.value).toBe(true)
  })
})

describe('the room of the system buttons', () => {
  it('keeps the width of the macOS buttons at the left on macOS', async () => {
    const frame = useWindowFrame(async () => createRuntime('mac').runtime)
    await flushPromises()

    expect(frame.windowEdge.value).toStrictEqual({ leftInset: MAC_BUTTONS_INSET, rightInset: 0 })
  })

  it('keeps no room on macOS in full screen, and keeps it again when the window leaves it', async () => {
    const { runtime } = createRuntime('mac')
    const state = { isFullscreen: true }
    const frame = useWindowFrame(async () => ({
      ...runtime,
      Window: { ...runtime.Window, IsFullscreen: async () => state.isFullscreen },
    }))
    await flushPromises()
    expect(frame.isFullscreen.value).toBe(true)
    expect(frame.windowEdge.value.leftInset).toBe(0)

    state.isFullscreen = false
    globalThis.dispatchEvent(new Event('resize'))
    await flushPromises()

    expect(frame.windowEdge.value.leftInset).toBe(MAC_BUTTONS_INSET)
  })

  it('keeps no room on Windows and Linux, where the window has no buttons of its own', () => {
    for (const platform of ['windows', 'linux'] as const) {
      expect(getWindowEdge(platform, false)).toStrictEqual({ leftInset: 0, rightInset: 0 })
    }
  })
})

describe('the corner radius of the window', () => {
  /** A macOS runtime that reports the version, and full screen when asked. */
  function createMacRuntime(version: string | undefined, isFullscreen = false): FrameRuntime {
    const { runtime } = createRuntime('mac')
    return {
      Window: { ...runtime.Window, IsFullscreen: async () => isFullscreen },
      System: {
        ...runtime.System,
        Environment: async () => ({ OSInfo: version === undefined ? {} : { Version: version } }),
      },
    }
  }

  it('takes the radius of macOS before 26 from the version in the runtime environment', async () => {
    const frame = useWindowFrame(async () => createMacRuntime('15.6.1'))
    await flushPromises()

    expect(frame.windowRadius.value).toBe(10)
  })

  it('takes the larger radius of macOS 26 from the version in the runtime environment', async () => {
    const frame = useWindowFrame(async () => createMacRuntime('26.6.2'))
    await flushPromises()

    expect(frame.windowRadius.value).toBe(16)
  })

  it('takes the larger radius when the version cannot be read', async () => {
    const frame = useWindowFrame(async () => createMacRuntime(undefined))
    await flushPromises()

    expect(frame.windowRadius.value).toBe(16)
  })

  it('takes the larger radius when the environment call fails', async () => {
    const runtime = createMacRuntime('15.0')
    const failing: FrameRuntime = {
      ...runtime,
      System: {
        ...runtime.System,
        Environment: async () => {
          throw new Error('no environment')
        },
      },
    }
    const frame = useWindowFrame(async () => failing)
    await flushPromises()

    expect(frame.windowRadius.value).toBe(16)
  })

  it('takes no radius in macOS full screen', async () => {
    const frame = useWindowFrame(async () => createMacRuntime('26.0', true))
    await flushPromises()

    expect(frame.windowRadius.value).toBe(0)
  })

  it('takes no radius on Windows and Linux', async () => {
    for (const platform of ['windows', 'linux'] as const) {
      const frame = useWindowFrame(async () => createRuntime(platform).runtime)
      await flushPromises()

      expect(frame.windowRadius.value).toBe(0)
    }
  })
})
