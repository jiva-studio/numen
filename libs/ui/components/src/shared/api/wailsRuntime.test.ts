import { afterEach, describe, expect, it, vi } from 'vitest'
import { openExternalUrl } from './wailsRuntime'

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
