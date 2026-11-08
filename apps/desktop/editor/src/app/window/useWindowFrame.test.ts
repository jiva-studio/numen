import { describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useWindowFrame, type FrameRuntime } from './useWindowFrame'

function createRuntime(platform: 'windows' | 'mac' | 'linux', maximised = false) {
  const state = { maximised }
  const calls = {
    Minimise: vi.fn(async () => undefined),
    ToggleMaximise: vi.fn(async () => {
      state.maximised = !state.maximised
    }),
    Close: vi.fn(async () => undefined),
  }
  const runtime: FrameRuntime = {
    Window: { ...calls, IsMaximised: async () => state.maximised },
    System: {
      IsWindows: () => platform === 'windows',
      IsMac: () => platform === 'mac',
      IsLinux: () => platform === 'linux',
    },
  }
  return { runtime, calls }
}

describe('useWindowFrame', () => {
  it('minimises, toggles maximise and closes through the Wails window', async () => {
    const { runtime, calls } = createRuntime('windows')
    const frame = useWindowFrame(async () => runtime)

    await frame.minimise()
    await frame.toggleMaximise()
    await frame.close()

    expect(calls.Minimise).toHaveBeenCalledOnce()
    expect(calls.ToggleMaximise).toHaveBeenCalledOnce()
    expect(calls.Close).toHaveBeenCalledOnce()
  })

  it('reports the maximised state of the window, also after it is toggled', async () => {
    const { runtime } = createRuntime('windows', true)
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()
    expect(frame.isMaximised.value).toBe(true)

    await frame.toggleMaximise()

    expect(frame.isMaximised.value).toBe(false)
  })

  it('draws the Windows buttons at the right on Windows', async () => {
    const frame = useWindowFrame(async () => createRuntime('windows').runtime)
    await flushPromises()

    expect(frame.controls.value).toBe('windows')
  })

  it('leaves the buttons to the system on macOS', async () => {
    const frame = useWindowFrame(async () => createRuntime('mac').runtime)
    await flushPromises()

    expect(frame.controls.value).toBe('native-mac')
  })

  it('draws the Windows buttons at the right on Linux', async () => {
    const frame = useWindowFrame(async () => createRuntime('linux').runtime)
    await flushPromises()

    expect(frame.controls.value).toBe('windows')
  })

  it('does nothing and stays unmaximised where Wails does not serve the page', async () => {
    const frame = useWindowFrame(async () => undefined)
    await flushPromises()

    await expect(frame.minimise()).resolves.toBeUndefined()
    await expect(frame.toggleMaximise()).resolves.toBeUndefined()
    await expect(frame.close()).resolves.toBeUndefined()
    expect(frame.isMaximised.value).toBe(false)
    expect(frame.controls.value).toBe('windows')
  })
})

describe('useWindowFrame when the read of the maximised state fails', () => {
  /** A runtime whose state read resolves once, then rejects, as a Wails window that went away mid-call. */
  function createFailingRuntime(maximised: boolean) {
    const failing = { value: false }
    const runtime: FrameRuntime = {
      Window: {
        ToggleMaximise: async () => {
          failing.value = true
        },
        IsMaximised: async () => {
          if (failing.value) throw new Error('the window state is not available')
          return maximised
        },
      },
    }
    return { runtime, failing }
  }

  it('resolves the toggle although the read of the state after it rejects', async () => {
    const { runtime } = createFailingRuntime(true)
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()

    await expect(frame.toggleMaximise()).resolves.toBeUndefined()
  })

  it('keeps the state it had when the read after the toggle rejects', async () => {
    const { runtime } = createFailingRuntime(true)
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()

    await frame.toggleMaximise().catch(() => undefined)

    expect(frame.isMaximised.value).toBe(true)
  })

  it('stays unmaximised when the first read of the state rejects', async () => {
    const runtime: FrameRuntime = {
      Window: {
        IsMaximised: async () => {
          throw new Error('the window state is not available')
        },
      },
    }
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()

    expect(frame.isMaximised.value).toBe(false)
  })

  it('keeps the state it had when the read after a resize rejects', async () => {
    const { runtime, failing } = createFailingRuntime(true)
    const frame = useWindowFrame(async () => runtime)
    await flushPromises()

    failing.value = true
    globalThis.dispatchEvent(new Event('resize'))
    await flushPromises()

    expect(frame.isMaximised.value).toBe(true)
  })
})
