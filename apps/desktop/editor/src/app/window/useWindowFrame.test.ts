import { describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useWindowFrame, type FrameRuntime } from './useWindowFrame'

/** The Wails runtime of a window on a platform, with the calls it receives recorded. */
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
