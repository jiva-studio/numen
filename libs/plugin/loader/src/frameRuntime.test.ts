import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createFrameRuntimeLoader,
  createWindowFrameControls,
  openExternalUrl,
} from './frameRuntime'

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
    const frame = createWindowFrameControls(async () => ({ Window: shell }))
    await frame.minimize()
    await frame.toggleMaximize()
    await frame.close()
    expect(shell.Minimise).toHaveBeenCalledOnce()
    expect(shell.ToggleMaximise).toHaveBeenCalledOnce()
    expect(shell.Close).toHaveBeenCalledOnce()
    expect(await frame.isMaximized()).toBe(true)
    expect(await frame.isFullscreen()).toBe(false)
    expect(shell.IsMaximised).toHaveBeenCalledOnce()
    expect(shell.IsFullscreen).toHaveBeenCalledOnce()
  })

  it.each(['darwin', 'linux', 'windows'] as const)('reads %s and its version', async (platform) => {
    const Environment = vi.fn(async () => ({ OS: platform, OSInfo: { Version: '26.0' } }))
    const frame = createWindowFrameControls(async () => ({ System: { Environment } }))
    expect(await frame.getEnvironment()).toEqual({ platform, version: '26.0' })
    await frame.isMaximized()
    await frame.toggleMaximize()
    expect(Environment).toHaveBeenCalledOnce()
  })

  it.each([undefined, {}, { Window: {}, System: {} }])(
    'handles absent shell controls',
    async (shell) => {
      const frame = createWindowFrameControls(async () => shell)
      expect(await frame.getEnvironment()).toEqual({ platform: 'windows', version: undefined })
      expect(await frame.isMaximized()).toBe(false)
      expect(await frame.isFullscreen()).toBe(false)
      await expect(frame.minimize()).resolves.toBeUndefined()
      await expect(frame.toggleMaximize()).resolves.toBeUndefined()
      await expect(frame.close()).resolves.toBeUndefined()
    },
  )
})

it('shares the pending shell import and its unavailable result', async () => {
  const load = createFrameRuntimeLoader()
  const pending = load()
  expect(load()).toBe(pending)
  await expect(pending).resolves.toBeUndefined()
  expect(load()).toBe(pending)
})

it('imports the shell separately for each loader', () => {
  expect(createFrameRuntimeLoader()()).not.toBe(createFrameRuntimeLoader()())
})
