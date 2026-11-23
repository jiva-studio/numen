/**
 * The settings tab is given the connections of the core: it reads them when a page asks, and words the
 * reason the core gives for a refused change. Every value here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { ConnectionInfo } from '@/entities/connection'
import type { MessageLog } from '@/shared/model/notices/messages'
import { WORDS } from '@/shared/ui/words'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createWindowSettingsPart } from '@/testing/windowSettings'

const connection: ConnectionInfo = {
  id: 'c00000001',
  typeId: 'a.b/x',
  name: 'Work',
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  isDetected: false,
  path: '',
  uses: [],
}

/** The calls of a window whose core answers none; a test answers the connections it reads. */
const api = createAnsweredApi({})

const createSettings = (
  commands = { getIds: () => ['cmd.one'], run: vi.fn(() => Promise.resolve('')) },
) => {
  const log = { getWriter: () => vi.fn() } as unknown as MessageLog
  const windowTabs = { handle: { openTab: vi.fn(() => Promise.resolve('')) } }
  return createWindowSettingsPart({
    api,
    rest: createSettingsStore(api.configuration.settings, SETTINGS_WORDS, vi.fn()),
    words: WORDS,
    log,
    windowTabs: windowTabs as unknown as WindowTabs,
    onSizeChanged: vi.fn(),
    getSettingsViews: () => new Map(),
    getPlatform: () => 'linux',
    health: { items: ref([]), commands },
  })
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('the connections in the settings tab', () => {
  it('are read when the page asks, and listed in the installation', async () => {
    vi.spyOn(api.configuration.connection, 'listConnectionTypes').mockResolvedValue({
      types: [],
      operationTitles: {},
    })
    const list = vi
      .spyOn(api.configuration.connection, 'listConnections')
      .mockResolvedValue([connection])
    const { connections: installed } = createSettings().configured.state.installation

    await installed.load()

    expect(installed.list.value).toStrictEqual([connection])
    expect(list).toHaveBeenCalledOnce()
  })

  it('give the reason in words when the core refuses a create', async () => {
    vi.spyOn(api.configuration.connection, 'createConnection').mockResolvedValue({
      ok: false,
      error: { error: 'nameTaken', message: 'Work' },
    })
    const { connections: installed } = createSettings().configured.state.installation

    const answer = await installed.edits.create({
      typeId: 'a.b/x',
      name: 'Work',
      fields: {},
      secrets: {},
    })

    expect(answer).toStrictEqual({ ok: false, error: 'Another connection has this name.' })
  })

  it('counts notes in the vault through listFiles', async () => {
    vi.spyOn(api.vault.file, 'listFiles').mockResolvedValue([
      { path: 'a.md', name: 'a.md', isFolder: false, isNote: true, type: 'markdown' },
      { path: 'b.png', name: 'b.png', isFolder: false, isNote: false, type: 'image' },
    ])
    const { connections: installed } = createSettings().configured.state.installation

    expect(await installed.countNotes()).toBe(1)
  })

  it('reloads appearance only when a settingbatch includes an app setting', async () => {
    vi.spyOn(api.configuration.settings, 'updateSettings').mockResolvedValue({
      written: '{}',
      version: 1n,
    })
    vi.spyOn(api.configuration.settings, 'getSettings').mockResolvedValue({
      written: '{}',
      version: 1n,
      path: '/numen.json',
      plugins: [],
      app: [],
    })
    const getAppearance = vi.spyOn(api.configuration.themes, 'getAppearance').mockResolvedValue({
      themes: [],
      applied: 'preset:numen',
      mode: 'system',
      sizes: { interfaceScale: 1, textScale: 1 },
      bounds: {
        interfaceScale: { min: 0.8, max: 2 },
        textScale: { min: 0.8, max: 2 },
      },
    })
    const { settings } = createSettings().configured.state.installation

    settings.write([{ at: ['plugin.a', 'key'], value: '1' }])
    await flushPromises()
    expect(getAppearance).not.toHaveBeenCalled()

    settings.write([
      { at: ['plugin.a', 'key'], value: '1' },
      { at: ['app', 'appearance', 'theme'], value: '"preset:numen"' },
    ])
    await flushPromises()
    expect(getAppearance).toHaveBeenCalledOnce()
  })

  it('checks and runs health commands and refreshes settings after plugin retry', async () => {
    const run = vi.fn(() => Promise.resolve(''))
    const page = createSettings({ getIds: () => ['cmd.one'], run })
    const { plugin } = page.configured.state.installation

    expect(plugin.health.canRunCommand('cmd.one')).toBe(true)
    expect(plugin.health.canRunCommand('cmd.two')).toBe(false)

    plugin.health.runCommand('cmd.one', { setting: 'source' })
    await flushPromises()
    expect(run).toHaveBeenCalledWith('cmd.one', undefined, undefined, { setting: 'source' })

    const getSettings = vi.spyOn(api.configuration.settings, 'getSettings').mockResolvedValue({
      written: '{}',
      version: 1n,
      path: '/numen.json',
      plugins: [],
      app: [],
    })
    vi.spyOn(api.configuration.settings, 'retryPlugin').mockResolvedValue({
      ok: true,
      value: null,
    })

    plugin.retryPlugin('plugin.a')
    await flushPromises()
    expect(getSettings).toHaveBeenCalledOnce()
  })
})
