import { afterEach, describe, expect, it, vi } from 'vitest'
import { createExternalOpener } from './externalOpener'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('opening an address from a plugin', () => {
  it('passes a web address to the opener', async () => {
    const open = vi.fn().mockResolvedValue(undefined)

    const result = await createExternalOpener(open)('https://numen.md/')

    expect(result).toEqual({ ok: true, value: null })
    expect(open).toHaveBeenCalledWith('https://numen.md/')
  })

  it('fails for any other address without calling the opener', async () => {
    const open = vi.fn()

    const result = await createExternalOpener(open)('file:///etc/passwd')

    expect(result).toEqual({
      ok: false,
      error: { error: 'notWebAddress', message: 'not a web address: file:///etc/passwd' },
    })
    expect(open).not.toHaveBeenCalled()
  })

  it('reports the failure of the opener', async () => {
    const open = vi.fn().mockRejectedValue(new Error('no browser'))

    const result = await createExternalOpener(open)('https://numen.md/')

    expect(result).toEqual({
      ok: false,
      error: { error: 'openFailed', message: expect.any(String) },
    })
  })
})
