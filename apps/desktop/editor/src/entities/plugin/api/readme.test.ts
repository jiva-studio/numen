import { afterEach, describe, expect, it, vi } from 'vitest'
import { readPluginReadme } from './readme'

describe('readPluginReadme', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns empty string when fetch is unavailable, fails, returns non-ok, or returns html', async () => {
    vi.stubGlobal('fetch', undefined)
    expect(await readPluginReadme('my-plugin')).toBe('')

    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    fetchMock.mockResolvedValueOnce({ ok: false, text: async () => 'Error body' })
    expect(await readPluginReadme('my-plugin')).toBe('')
    expect(fetchMock).toHaveBeenCalledWith('/plugin/my-plugin/README.md')

    fetchMock.mockResolvedValueOnce({
      ok: true,
      headers: new Headers({ 'content-type': 'text/html; charset=utf-8' }),
      text: async () => '<html></html>',
    })
    expect(await readPluginReadme('my-plugin')).toBe('')

    fetchMock.mockResolvedValueOnce({
      ok: true,
      headers: new Headers({ 'content-type': 'text/markdown' }),
      text: async () => '# Hello',
    })
    expect(await readPluginReadme('my-plugin')).toBe('# Hello')

    fetchMock.mockResolvedValueOnce({
      ok: true,
      headers: new Headers(),
      text: async () => '# Bare',
    })
    expect(await readPluginReadme('my-plugin')).toBe('# Bare')

    fetchMock.mockRejectedValueOnce(new Error('network error'))
    expect(await readPluginReadme('my-plugin')).toBe('')
  })
})
