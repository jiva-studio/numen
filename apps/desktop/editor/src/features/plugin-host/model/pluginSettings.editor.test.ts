/**
 * What a plugin reads of the editor settings: the two booleans of the application settings the
 * window holds, and nothing before they have arrived. The settings come through the real store over
 * the transport of a test core. Every value here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { createSettingsStore, SETTINGS_WORDS, type PluginSettingInfo } from '@/entities/settings'
import { createSettingsCoreApi, type SettingsCore } from '@/testing/settingsWire'
import { createPluginSettingsAccessor } from './pluginSettings'

const PLUGIN = 'md.numen.sample'

const DECLARATION: readonly PluginSettingInfo[] = [
  {
    key: 'editor.fontSize',
    type: 'range',
    title: 'Font size',
    description: '',
    default: 14,
    min: 10,
    max: 32,
    step: 1,
    isShown: true,
  },
  {
    key: 'editor.lineNumbers',
    type: 'boolean',
    title: 'Line numbers',
    description: '',
    default: false,
    isShown: true,
  },
  {
    key: 'editor.softWrap',
    type: 'boolean',
    title: 'Soft wrap',
    description: '',
    default: true,
    isShown: true,
  },
]

const createWindow = (file: object, core: Partial<SettingsCore> = {}) => {
  const api = createSettingsCoreApi({
    getSettings: () =>
      Promise.resolve({ written: JSON.stringify(file), path: '/numen.json', app: DECLARATION }),
    updateSettings: () => Promise.resolve(),
    ...core,
  })
  const store = createSettingsStore(api.configuration.settings, SETTINGS_WORDS, vi.fn())
  const reportError = vi.fn()
  return { store, reportError, accessor: createPluginSettingsAccessor(store, reportError) }
}

describe('the editor settings a plugin reads', () => {
  it('are undefined until the window has loaded its settings', () => {
    const { accessor } = createWindow({})

    expect(accessor.getEditorSettings()).toBeUndefined()
  })

  it('are the values of the file once the settings are loaded', async () => {
    const { store, accessor } = createWindow({
      app: { editor: { lineNumbers: true, softWrap: false } },
    })

    await store.start()

    expect(accessor.getEditorSettings()).toStrictEqual({ lineNumbers: true, softWrap: false })
  })

  it('are the declared defaults where the file holds no value', async () => {
    const { store, accessor } = createWindow({})

    await store.start()

    expect(accessor.getEditorSettings()).toStrictEqual({ lineNumbers: false, softWrap: true })
  })

  it('stay undefined where the first read of the settings failed', async () => {
    const { store, accessor } = createWindow(
      {},
      { getSettings: () => Promise.reject(createLostConnection('the core is busy')) },
    )

    await expect(store.start()).rejects.toThrow('the core is busy')

    expect(accessor.getEditorSettings()).toBeUndefined()
  })

  it('show a value written in the settings', async () => {
    const { store, accessor } = createWindow({})
    await store.start()

    await store.write.writeSetting(['app', 'editor', 'softWrap'], false)

    expect(accessor.getEditorSettings()).toStrictEqual({ lineNumbers: false, softWrap: false })
  })

  it('carry nothing but the two booleans, though the font size is a setting of the same group', async () => {
    const { store, accessor } = createWindow({ app: { editor: { fontSize: 20 } } })

    await store.start()

    expect(Object.keys(accessor.getEditorSettings() ?? {}).sort()).toStrictEqual([
      'lineNumbers',
      'softWrap',
    ])
  })
})

describe('the handler of the editor settings', () => {
  it('is called with the values when they first arrive', async () => {
    const { store, accessor } = createWindow({ app: { editor: { lineNumbers: true } } })
    const handler = vi.fn()
    accessor.watchEditorSettings(PLUGIN, handler)

    await store.start()

    await vi.waitFor(() =>
      expect(handler).toHaveBeenCalledExactlyOnceWith({ lineNumbers: true, softWrap: true }),
    )
  })

  it('is not called while the settings have not arrived', async () => {
    const { store, accessor } = createWindow(
      {},
      { getSettings: () => Promise.reject(createLostConnection('the core is busy')) },
    )
    const handler = vi.fn()
    accessor.watchEditorSettings(PLUGIN, handler)

    await expect(store.start()).rejects.toThrow('the core is busy')
    await Promise.resolve()

    expect(handler).not.toHaveBeenCalled()
  })

  it('is called with the new values when the wrap changes', async () => {
    const { store, accessor } = createWindow({})
    await store.start()
    const handler = vi.fn()
    accessor.watchEditorSettings(PLUGIN, handler)

    await store.write.writeSetting(['app', 'editor', 'softWrap'], false)

    await vi.waitFor(() =>
      expect(handler).toHaveBeenLastCalledWith({ lineNumbers: false, softWrap: false }),
    )
  })

  it('is called with the new values when the line numbers change', async () => {
    const { store, accessor } = createWindow({})
    await store.start()
    const handler = vi.fn()
    accessor.watchEditorSettings(PLUGIN, handler)

    await store.write.writeSetting(['app', 'editor', 'lineNumbers'], true)

    await vi.waitFor(() =>
      expect(handler).toHaveBeenLastCalledWith({ lineNumbers: true, softWrap: true }),
    )
  })

  it('is not called for a change of the font size, which a plugin reads as a token', async () => {
    const { store, accessor } = createWindow({})
    await store.start()
    const handler = vi.fn()
    accessor.watchEditorSettings(PLUGIN, handler)

    await store.write.writeSetting(['app', 'editor', 'fontSize'], 22)
    await store.write.writeSetting(['app', 'editor', 'softWrap'], false)

    await vi.waitFor(() => expect(handler).toHaveBeenCalled())
    expect(handler).toHaveBeenCalledExactlyOnceWith({ lineNumbers: false, softWrap: false })
  })

  it('is not called again once the subscription is stopped', async () => {
    const { store, accessor } = createWindow({})
    await store.start()
    const handler = vi.fn()
    const stop = accessor.watchEditorSettings(PLUGIN, handler)
    stop.dispose()

    await store.write.writeSetting(['app', 'editor', 'softWrap'], false)
    await Promise.resolve()

    expect(handler).not.toHaveBeenCalled()
  })

  it('reports an error it throws against the plugin and keeps following', async () => {
    const { store, accessor, reportError } = createWindow({})
    await store.start()
    const failure = new Error('the handler broke')
    const handler = vi.fn().mockRejectedValueOnce(failure).mockResolvedValue(undefined)
    accessor.watchEditorSettings(PLUGIN, handler)

    await store.write.writeSetting(['app', 'editor', 'softWrap'], false)
    await vi.waitFor(() => expect(reportError).toHaveBeenCalledWith(PLUGIN, expect.anything()))
    await store.write.writeSetting(['app', 'editor', 'lineNumbers'], true)

    await vi.waitFor(() => expect(handler).toHaveBeenCalledTimes(2))
  })
})
