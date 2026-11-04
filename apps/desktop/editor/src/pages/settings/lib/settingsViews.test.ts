import { afterEach, describe, expect, it, vi } from 'vitest'
import { getRegisteredViews, getSettingValues } from './settingsViews'
import { getShownConsent, hasPluginSettings } from './pluginSettings'
import { fetchPluginReadme } from './readme'
import { createPluginInfo } from '@/testing/installation'

describe('getRegisteredViews', () => {
  const declared = [
    { id: 'top', title: 'Top' },
    { id: 'floor', title: 'Floor' },
    { id: 'missing', title: 'Missing' },
  ]
  const registered = new Map([
    ['top', { component: 'T' }],
    ['floor', { component: 'F' }],
  ])

  it('returns the registered views in declared order', () => {
    expect(getRegisteredViews(declared, registered)).toStrictEqual([
      { id: 'top', component: 'T' },
      { id: 'floor', component: 'F' },
    ])
  })

  it('returns nothing while no view is registered', () => {
    expect(getRegisteredViews(declared, new Map())).toStrictEqual([])
  })
})

describe('getSettingValues', () => {
  it('reads every declared setting by key', () => {
    const plugin = createPluginInfo('p', 'P', {
      settings: [
        { key: 'a', type: 'number', title: 'A', description: '', default: 1, palette: false },
        { key: 'b', type: 'string', title: 'B', description: '', default: '', palette: false },
      ],
    })
    expect(getSettingValues(plugin, (key) => (key === 'a' ? 5 : undefined))).toStrictEqual({
      a: 5,
      b: '',
    })
  })
})

describe('hasPluginSettings and getShownConsent', () => {
  it('detects settings, settingsViews, and visible secrets', () => {
    const empty = createPluginInfo('p', 'P')
    expect(hasPluginSettings(empty, () => undefined)).toBe(false)

    const withSetting = createPluginInfo('p', 'P', {
      settings: [{ key: 'a', type: 'string', title: 'A', description: '', default: '' }],
    })
    expect(hasPluginSettings(withSetting, () => undefined)).toBe(true)

    const withView = createPluginInfo('p', 'P', {
      settingsViews: [{ id: 'v', title: 'V' }],
    })
    expect(hasPluginSettings(withView, () => undefined)).toBe(true)

    const withHiddenSecretOnly = createPluginInfo('p', 'P', {
      settings: [],
      secrets: [
        { name: 's1', title: 'S1', description: '', when: { setting: 'mode', is: ['on'] } },
      ],
    })
    expect(hasPluginSettings(withHiddenSecretOnly, () => 'off')).toBe(false)

    const withNoSettingsSecretsOnly = createPluginInfo('p', 'P', {
      settings: [],
      secrets: [{ name: 's1', title: 'S1', description: '' }],
    })
    expect(hasPluginSettings(withNoSettingsSecretsOnly, () => undefined)).toBe(true)
  })

  it('computes shown consent from plugin consent and aiConsent setting', () => {
    const nonePlugin = createPluginInfo('p', 'P', { consent: 'none' })
    expect(getShownConsent(nonePlugin, (k) => (k === 'aiConsent' ? true : undefined))).toBe('none')

    const reqPlugin = createPluginInfo('p', 'P', { consent: 'requested' })
    expect(getShownConsent(reqPlugin, () => undefined)).toBe('requested')
    expect(getShownConsent(reqPlugin, (k) => (k === 'aiConsent' ? true : false))).toBe('allowed')
    expect(getShownConsent(reqPlugin, (k) => (k === 'aiConsent' ? false : true))).toBe('denied')
  })
})

describe('fetchPluginReadme', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns empty string when fetch is unavailable, fails, returns non-ok, or returns html', async () => {
    vi.stubGlobal('fetch', undefined)
    expect(await fetchPluginReadme('my-plugin')).toBe('')

    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    fetchMock.mockResolvedValueOnce({ ok: false, text: async () => 'Error body' })
    expect(await fetchPluginReadme('my-plugin')).toBe('')
    expect(fetchMock).toHaveBeenCalledWith('/plugin/my-plugin/README.md')

    fetchMock.mockResolvedValueOnce({
      ok: true,
      headers: { get: (h: string) => (h === 'content-type' ? 'text/html; charset=utf-8' : '') },
      text: async () => '<html></html>',
    })
    expect(await fetchPluginReadme('my-plugin')).toBe('')

    fetchMock.mockResolvedValueOnce({
      ok: true,
      headers: undefined,
      text: async () => '# Hello',
    })
    expect(await fetchPluginReadme('my-plugin')).toBe('# Hello')

    fetchMock.mockRejectedValueOnce(new Error('network error'))
    expect(await fetchPluginReadme('my-plugin')).toBe('')
  })
})
