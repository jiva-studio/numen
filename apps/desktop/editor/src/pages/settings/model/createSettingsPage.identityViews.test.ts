/**
 * The settings tab lists the views the plugins registered for the page "Identity and licenses": the
 * declared views of that page, in the order of the plugins, that the frontend of the plugin registered.
 * Every value here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import type { ViewComponent } from '../types'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { MessageLog } from '@/shared/model/notices/messages'
import { WORDS } from '@/shared/ui/words'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createWindowSettingsPart } from '@/testing/windowSettings'

const api = createAnsweredApi({})

const createPlugin = (id: string, views: readonly { id: string; page: string }[]) => ({
  id,
  name: id,
  version: '1.0.0',
  description: '',
  isActive: true,
  isOffByDefault: false,
  capabilities: [],
  hasFrontend: true,
  hasBackend: false,
  readers: [],
  settingsViews: views.map((one) => ({ id: one.id, title: one.id, page: one.page })),
  settingGroups: [],
  contributions: [],
  settings: [],
})

const createSettings = async (
  registered: Readonly<Record<string, ReadonlyMap<string, ViewComponent>>>,
  plugins: readonly ReturnType<typeof createPlugin>[],
) => {
  vi.spyOn(api.configuration.settings, 'getSettings').mockResolvedValue({
    written: '{}',
    version: 1n,
    path: '/numen.json',
    plugins: [...plugins],
    app: [],
    appGroups: [],
  })
  const rest = createSettingsStore(api.configuration.settings, SETTINGS_WORDS, vi.fn())
  const part = createWindowSettingsPart({
    api,
    rest,
    words: WORDS,
    log: { getWriter: () => vi.fn() } as unknown as MessageLog,
    windowTabs: { handle: { openTab: vi.fn(() => Promise.resolve('')) } } as unknown as WindowTabs,
    onSizeChanged: vi.fn(),
    getSettingsViews: (pluginId) => registered[pluginId] ?? new Map(),
    getPlatform: () => 'linux',
    health: {
      items: ref([]),
      commands: { getIds: () => [], run: vi.fn(() => Promise.resolve('')) },
    },
  })
  await rest.refresh()
  return part.configured.state.installation.plugin
}

describe('getIdentityViews of the installation', () => {
  const SUBSCRIPTION = { name: 'Subscription' }
  const VAULTS = { name: 'Vaults' }
  const CARD = { name: 'Card' }

  it('lists the registered views of the page of the identity in the order of the plugins', async () => {
    const plugin = await createSettings(
      {
        'a.sync': new Map([
          ['vaults', { component: VAULTS }],
          ['subscription', { component: SUBSCRIPTION }],
        ]),
        'b.cards': new Map([['card', { component: CARD }]]),
      },
      [
        createPlugin('a.sync', [
          { id: 'vaults', page: '' },
          { id: 'subscription', page: 'identity' },
        ]),
        createPlugin('b.cards', [{ id: 'card', page: 'identity' }]),
      ],
    )

    expect(plugin.getIdentityViews()).toStrictEqual([
      { pluginId: 'a.sync', id: 'subscription', component: SUBSCRIPTION },
      { pluginId: 'b.cards', id: 'card', component: CARD },
    ])
  })

  it('leaves out a declared view the plugin did not register', async () => {
    const plugin = await createSettings({ 'a.sync': new Map() }, [
      createPlugin('a.sync', [{ id: 'subscription', page: 'identity' }]),
    ])

    expect(plugin.getIdentityViews()).toStrictEqual([])
  })

  it('lists nothing while no plugin declares a view of the page', async () => {
    const plugin = await createSettings(
      { 'a.sync': new Map([['vaults', { component: VAULTS }]]) },
      [createPlugin('a.sync', [{ id: 'vaults', page: '' }])],
    )

    expect(plugin.getIdentityViews()).toStrictEqual([])
  })
})
