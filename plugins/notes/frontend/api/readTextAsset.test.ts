import { afterEach, describe, expect, it, vi } from 'vitest'
import { readTextAsset } from './readTextAsset'

describe('reading a text file the window serves', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('answers the text of the file', async () => {
    vi.stubGlobal('fetch', async () => new Response('# Tides\n'))

    expect(await readTextAsset('Nature/Tides.md')).toStrictEqual({ ok: true, value: '# Tides\n' })
  })

  it('answers why the file could not be read', async () => {
    vi.stubGlobal('fetch', async () => new Response('', { status: 404 }))

    expect(await readTextAsset('Nature/Gone.md')).toStrictEqual({
      ok: false,
      error: 'Nature/Gone.md: 404',
    })
  })
})
