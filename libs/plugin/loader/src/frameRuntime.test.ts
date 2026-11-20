import { afterEach, describe, expect, it, vi } from 'vitest'
import { loadWindowFrameControls, openExternalUrl } from './frameRuntime'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('opening an address in the system browser', () => {
  it('passes it to the runtime where the page has one', async () => {
    const OpenURL = vi.fn()
    const open = vi.spyOn(window, 'open').mockReturnValue(null)

    await openExternalUrl('https://example.org/', async () => ({ Browser: { OpenURL } }))

    expect(OpenURL).toHaveBeenCalledWith('https://example.org/')
    expect(open).not.toHaveBeenCalled()
  })

  it('opens a browsing context where the page has no runtime', async () => {
    const open = vi.spyOn(window, 'open').mockReturnValue(null)

    await openExternalUrl('https://example.org/', async () => undefined)
    await openExternalUrl('https://example.org/', async () => ({}))

    expect(open).toHaveBeenCalledTimes(2)
    expect(open).toHaveBeenCalledWith('https://example.org/', '_blank')
  })

  it('rejects where the runtime fails to open the address', async () => {
    const OpenURL = vi.fn().mockRejectedValue(new Error('no browser'))

    await expect(
      openExternalUrl('https://example.org/', async () => ({ Browser: { OpenURL } })),
    ).rejects.toThrow('no browser')
  })
})

describe('desktop frame controls', () => {
  it('forwards window actions and state queries to the runtime', async () => {
    const shell = {
      Minimise: vi.fn(async () => undefined),
      ToggleMaximise: vi.fn(async () => undefined),
      Close: vi.fn(async () => undefined),
      IsMaximised: vi.fn(async () => true),
      IsFullscreen: vi.fn(async () => false),
    }
    const frame = await loadWindowFrameControls(async () => ({ Window: shell }))
    await frame?.minimize()
    await frame?.toggleMaximize()
    await frame?.close()
    expect(shell.Minimise).toHaveBeenCalledOnce()
    expect(shell.ToggleMaximise).toHaveBeenCalledOnce()
    expect(shell.Close).toHaveBeenCalledOnce()
    expect(await frame?.isMaximized()).toBe(true)
    expect(await frame?.isFullscreen()).toBe(false)
    expect(shell.IsMaximised).toHaveBeenCalledOnce()
    expect(shell.IsFullscreen).toHaveBeenCalledOnce()
  })

  it('identifies macOS and Linux and reads the system version', async () => {
    const mac = await loadWindowFrameControls(async () => ({
      System: { IsMac: () => true, Environment: async () => ({ OSInfo: { Version: '26.0' } }) },
    }))
    const linux = await loadWindowFrameControls(async () => ({ System: { IsLinux: () => true } }))
    expect(mac?.getPlatform()).toBe('darwin')
    expect(await mac?.getVersion()).toBe('26.0')
    expect(linux?.getPlatform()).toBe('linux')
  })

  it('handles missing shell modules and window controls', async () => {
    expect(await loadWindowFrameControls(async () => undefined)).toBeUndefined()
    const frame = await loadWindowFrameControls(async () => ({}))
    expect(frame?.getPlatform()).toBe('windows')
    expect(await frame?.getVersion()).toBeUndefined()
    expect(await frame?.isMaximized()).toBe(false)
    expect(await frame?.isFullscreen()).toBe(false)
    await expect(frame?.minimize()).resolves.toBeUndefined()
    await expect(frame?.toggleMaximize()).resolves.toBeUndefined()
    await expect(frame?.close()).resolves.toBeUndefined()
  })
})
