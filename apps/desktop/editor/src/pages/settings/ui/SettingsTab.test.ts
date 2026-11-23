/**
 * The settings tab rendered in a document.
 *
 * The tests check that a row writes through the same value as the command of that
 * name, that a setting read from the file is written back to the same place,
 * and that what the file holds is shown as selected.
 *
 * Every value here is invented.
 */
import type { JsonObject } from '@/shared/api/json'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { NumberField, Select, Switch } from '@numen/ui'

import type { PluginInfo, PluginSettingInfo, SettingEdit } from '@/entities/settings'
import type { LicenseInfo } from '@/entities/license'
import { getSettingAt as at } from '@/pages/settings/testing/settingAt'
import {
  createInstallation,
  createSettingsState,
  type InstallationOverrides,
} from '@/pages/settings/testing/installation'
import { asFailure, asValue } from '@numen/result'
import SettingsTab from './SettingsTab.vue'
import SettingRow from './setting-row/SettingRow.vue'
import type { Installation } from '../types'
import { WORDS as words } from '../words'

/** The settings of the shell as the core declares them and resolves them. */
const SHELL: readonly PluginSettingInfo[] = [
  {
    key: 'appearance.theme',
    type: 'enum',
    title: 'Theme',
    description: '',
    default: 'preset:numen',
    optionsFrom: 'themes',
    preview: true,
    activeOptions: [
      { value: 'preset:numen', title: 'numen', detail: '' },
      { value: 'plugin:md.example.themes/dusk', title: 'Dusk', detail: '' },
      { value: 'mine:sea', title: 'sea', detail: '' },
    ],
    isShown: true,
    value: 'preset:numen',
  },
  {
    key: 'naming.syncTitleAndFilename',
    type: 'boolean',
    title: 'Sync title and filename',
    description: '',
    default: true,
    isShown: true,
    value: true,
  },
]

const EDITOR: readonly PluginSettingInfo[] = [
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
    value: 14,
  },
  {
    key: 'editor.lineNumbers',
    type: 'boolean',
    title: 'Line numbers',
    description: '',
    default: false,
    isShown: true,
    value: false,
  },
]

const SEARCH: readonly PluginSettingInfo[] = [
  {
    key: 'search.reranking',
    type: 'connections',
    title: 'Reranking',
    description: 'Services that order the results of a query, tried in order',
    default: undefined,
    operations: ['llm.rerank'],
    max: 5,
    isShown: true,
  },
  {
    key: 'search.defaultLimit',
    type: 'number',
    title: 'Default results',
    description: '',
    default: 20,
    min: 1,
    max: 100,
    step: 1,
    isShown: true,
    value: 20,
  },
  {
    key: 'search.maxLimit',
    type: 'number',
    title: 'Most results',
    description: '',
    default: 100,
    min: 1,
    max: 1000,
    step: 1,
    isShown: true,
    value: 100,
  },
]

const createTab = (
  file: JsonObject = {},
  plugins: readonly PluginInfo[] = [],
  app: readonly PluginSettingInfo[] = SHELL,
  overrides: InstallationOverrides = {},
) => {
  const content: SettingEdit[] = []
  const installation: Installation = createInstallation({
    getSetting: (path) => at(file, path),
    write: (response) => {
      content.push(...response)
    },
    plugins: ref(plugins),
    app: ref(app),
    ...overrides,
  })
  const initialSearchQuery = ref('')
  const tab = mount(SettingsTab, {
    props: { state: createSettingsState(installation, { initialSearchQuery }) },
    attachTo: document.body,
  })
  mountedWrappers.push(tab)
  return {
    writes: content,
    recordedWrites: content,
    tab,
    initialSearchQuery,
  }
}

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
})

/** Every tab this file drew, put away between one test and the next. */
const mountedWrappers: { unmount: () => void }[] = []

afterEach(() => {
  vi.unstubAllGlobals()
  while (mountedWrappers.length) mountedWrappers.pop()?.unmount()
})

type Tab = ReturnType<typeof createTab>['tab']

const findSelect = (tab: Tab, title: string) => {
  const found = tab.findAllComponents(Select).find((one) => one.props('name') === title)
  if (!found) throw new Error(`no select is drawn for ${title}`)
  return found
}

const getMenuItems = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__item .menu__text')).map(
    (one) => one.textContent?.trim() ?? '',
  )

const getMenuGroups = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__group-name')).map(
    (one) => one.textContent?.trim() ?? '',
  )

const chooseItem = async (words: string) => {
  const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item'))
  rows.find((one) => one.querySelector('.menu__text')?.textContent?.trim() === words)?.click()
  await nextTick()
}

const openSystemSettings = async (tab: Tab) => {
  await tab.get('button[aria-label="System"]').trigger('click')
  await nextTick()
}

const openAppearanceSettings = async (tab: Tab) => {
  await tab.get('button[aria-label="Appearance"]').trigger('click')
  await nextTick()
}

const openEditorSettings = async (tab: Tab) => {
  await tab.get('button[aria-label="Editor"]').trigger('click')
  await nextTick()
}

const openSearchSettings = async (tab: Tab) => {
  await tab.get('button[aria-label="Search"]').trigger('click')
  await nextTick()
}

const getTileNames = (tab: Tab) =>
  tab
    .get('.plugins-page')
    .findAll('button')
    .map((one) => one.attributes('aria-label'))

const openPlugin = async (tab: Tab, name: string) => {
  await tab.get(`button[aria-label="${name}"]`).trigger('click')
  await nextTick()
}

const goBack = async (tab: Tab) => {
  await tab.get('button[aria-label="Back"]').trigger('click')
  await nextTick()
}

const isPluginPage = (tab: Tab) => tab.find('[role="tablist"]').exists()

const isList = (tab: Tab) => tab.find('button[aria-label="System"]').exists()

describe('the settings tab', () => {
  it('draws the System tile at the top of the list', () => {
    const { tab } = createTab()
    const system = tab.get('button[aria-label="System"]')
    expect(getTileNames(tab)[0]).toBe(words.systemSettings)
    expect(system.text()).toContain(words.systemSettingsDetail)
  })

  it('draws the Appearance, Editor, Search, Connections and Licenses tiles after System and the plugin rows below them', () => {
    const samplePlugin: PluginInfo = {
      id: 'test.transcription',
      name: 'Transcription Plugin',
      version: '1.0.0',
      description: 'Speech recognition engine',
      isActive: true,
      capabilities: [],
      hasFrontend: false,
      hasBackend: false,
      settings: [],
    }
    const { tab } = createTab({}, [samplePlugin])
    expect(getTileNames(tab)).toStrictEqual([
      'System',
      'Appearance',
      'Editor',
      'Search',
      'Connections',
      'Licenses',
      'Transcription Plugin',
    ])
  })

  it('opens on the theme the core resolved, in one list with the themes of plugins', async () => {
    const { tab } = createTab()
    await openAppearanceSettings(tab)
    const theme = findSelect(tab, 'Theme')
    expect(theme.text()).toContain('numen')

    await theme.get('[data-slot="select"]').trigger('click')
    await nextTick()
    await nextTick()
    expect(getMenuItems()).toStrictEqual(['numen', 'Dusk', 'sea'])
    expect(getMenuGroups()).toStrictEqual([])
  })

  it('writes a theme chosen under the owner app', async () => {
    const { tab, writes } = createTab()
    await openAppearanceSettings(tab)
    await findSelect(tab, 'Theme').get('[data-slot="select"]').trigger('click')
    await nextTick()
    await nextTick()
    await chooseItem('Dusk')
    expect(writes).toStrictEqual([
      { at: ['app', 'appearance', 'theme'], value: '"plugin:md.example.themes/dusk"' },
    ])
  })

  it('turns the naming switch by writing it under the owner app', async () => {
    const { tab, writes } = createTab()
    await openSystemSettings(tab)
    tab.getComponent(Switch).vm.$emit('update:modelValue', false)
    expect(writes).toStrictEqual([
      { at: ['app', 'naming', 'syncTitleAndFilename'], value: 'false' },
    ])
  })

  it('opens the Editor page from its tile with the editor settings and nothing else', async () => {
    const { tab } = createTab({}, [], [...SHELL, ...EDITOR])

    await openEditorSettings(tab)

    expect(tab.get('[aria-current="page"]').text()).toBe('Editor')
    expect(tab.findAllComponents(SettingRow).map((one) => one.props('name'))).toStrictEqual([
      'Font size',
      'Line numbers',
    ])
  })

  it('opens the Search page from its tile with the search settings and nothing else', async () => {
    const { tab } = createTab({}, [], [...SHELL, ...EDITOR, ...SEARCH])

    await openSearchSettings(tab)

    expect(tab.get('[aria-current="page"]').text()).toBe('Search')
    expect(tab.findAllComponents(SettingRow).map((one) => one.props('name'))).toStrictEqual([
      'Reranking',
      'Default results',
      'Most results',
    ])
    expect(tab.find('.system-detail').exists()).toBe(true)
  })

  it('keeps the search settings off the System and the Editor pages', async () => {
    const { tab } = createTab({}, [], [...SHELL, ...EDITOR, ...SEARCH])

    await openSystemSettings(tab)
    expect(tab.findAllComponents(SettingRow).map((one) => one.props('name'))).toStrictEqual([
      'Sync title and filename',
    ])

    await goBack(tab)
    await openEditorSettings(tab)
    expect(tab.findAllComponents(SettingRow).map((one) => one.props('name'))).toStrictEqual([
      'Font size',
      'Line numbers',
    ])
  })

  it('writes the default results set on the Search page under the owner app', async () => {
    const { tab, writes } = createTab({}, [], [...SHELL, ...SEARCH])
    await openSearchSettings(tab)

    tab.getComponent(NumberField).vm.$emit('commit', 50)

    expect(writes).toStrictEqual([{ at: ['app', 'search', 'defaultLimit'], value: '50' }])
  })

  it('keeps the editor settings off the System page', async () => {
    const { tab } = createTab({}, [], [...SHELL, ...EDITOR])

    await openSystemSettings(tab)

    expect(tab.findAllComponents(SettingRow).map((one) => one.props('name'))).toStrictEqual([
      'Sync title and filename',
    ])
  })

  it('turns line numbers by writing them under the owner app', async () => {
    const { tab, writes } = createTab({}, [], [...SHELL, ...EDITOR])
    await openEditorSettings(tab)

    tab.getComponent(Switch).vm.$emit('update:modelValue', true)

    expect(writes).toStrictEqual([{ at: ['app', 'editor', 'lineNumbers'], value: 'true' }])
  })

  it('names the file the settings stand in, and offers no tab to edit it', async () => {
    const { tab } = createTab()
    await openSystemSettings(tab)
    expect(tab.get('.system-detail__file').text()).toBe('/numen.json')
    expect(tab.findAll('button').map((one) => one.text())).not.toContain('Edit')
  })

  it('carries no pencil, and no editor spliced under a row', async () => {
    const { tab } = createTab()
    await openSystemSettings(tab)
    expect(tab.findAll('.cm-editor')).toHaveLength(0)
    expect(tab.text()).not.toContain('JSON5')
  })

  it('switches to dedicated system detail and returns on back', async () => {
    const { tab } = createTab()
    expect(tab.find('.system-detail').exists()).toBe(false)
    await openSystemSettings(tab)
    expect(tab.find('.system-detail').exists()).toBe(true)
    expect(tab.get('[aria-current="page"]').text()).toBe(words.systemSettings)

    await goBack(tab)
    expect(tab.find('.system-detail').exists()).toBe(false)
    expect(isList(tab)).toBe(true)
  })

  it('switches to full-page plugin detail when a plugin is selected, and returns on back', async () => {
    const samplePlugin: PluginInfo = {
      id: 'md.numen.engines.ocr',
      name: 'Document OCR',
      version: '1.0.0',
      description: 'Optical character recognition',
      isActive: true,
      capabilities: [],
      hasFrontend: false,
      hasBackend: false,
      settings: [],
    }
    const { tab } = createTab({}, [samplePlugin])

    await openPlugin(tab, 'Document OCR')

    expect(isList(tab)).toBe(false)
    expect(isPluginPage(tab)).toBe(true)
    expect(tab.get('h2').text()).toBe('Document OCR')

    await goBack(tab)

    expect(isPluginPage(tab)).toBe(false)
    expect(isList(tab)).toBe(true)
  })

  it('opens the Connections page from the rows of a plugin that has none', async () => {
    const proofreading: PluginInfo = {
      id: 'md.test.proofreading',
      name: 'Test Proofreading',
      version: '1.0.0',
      description: '',
      isActive: true,
      capabilities: [],
      hasFrontend: false,
      hasBackend: true,
      settings: [
        {
          key: 'connections',
          type: 'connections',
          title: 'Connections',
          description: '',
          default: undefined,
          operations: ['llm.complete'],
          max: 5,
          isShown: true,
        },
      ],
    }
    const { tab } = createTab({}, [proofreading])
    await openPlugin(tab, 'Test Proofreading')
    expect(tab.find('.connections-page').exists()).toBe(false)

    const open = tab.findAll('button').find((one) => one.text() === 'Open Connections')
    await open!.trigger('click')
    await nextTick()

    expect(tab.find('.connections-page').exists()).toBe(true)
    expect(isPluginPage(tab)).toBe(false)
  })

  it('keeps the cleared search query when navigating into a plugin detail and back', async () => {
    const samplePlugin: PluginInfo = {
      id: 'md.numen.views.agent-chat',
      name: 'Agent Chat',
      version: '1.0.0',
      description: 'Interactive chat tab',
      isActive: true,
      capabilities: ['agent'],
      hasFrontend: true,
      hasBackend: false,
      settings: [],
    }
    const { tab, initialSearchQuery } = createTab({}, [samplePlugin])
    initialSearchQuery.value = 'feature:agent'
    await nextTick()

    const searchInput = tab.get('input[type="search"]')
    expect((searchInput.element as HTMLInputElement).value).toBe('feature:agent')

    await searchInput.setValue('')
    await nextTick()
    expect(initialSearchQuery.value).toBe('')

    await openPlugin(tab, 'Agent Chat')
    expect(isPluginPage(tab)).toBe(true)

    await goBack(tab)

    const restoredInput = tab.get('input[type="search"]')
    expect((restoredInput.element as HTMLInputElement).value).toBe('')
  })
  describe('the toolbar', () => {
    const OCR: PluginInfo = {
      id: 'md.example.ocr',
      name: 'Page Reader',
      version: '1.0.0',
      description: 'Reads the text of scanned pages',
      isActive: true,
      capabilities: [],
      hasFrontend: false,
      hasBackend: false,
      settings: [],
    }

    const getToolbar = (tab: Tab) => tab.get('.settings-toolbar')
    const getPath = (tab: Tab) => getToolbar(tab).get('nav[aria-label="Path"]')
    const getSegments = (tab: Tab) =>
      getPath(tab)
        .findAll('button')
        .map((one) => one.text())
    const getCurrent = (tab: Tab) => getPath(tab).get('[aria-current="page"]').text()

    it('shows only Settings on the list, with a disabled back button and no action', () => {
      const { tab } = createTab({}, [OCR])

      expect(getSegments(tab)).toStrictEqual([])
      expect(getCurrent(tab)).toBe('Settings')
      expect(getToolbar(tab).get('button[aria-label="Back"]').attributes('disabled')).toBeDefined()
      expect(getToolbar(tab).find('.settings-toolbar__actions').exists()).toBe(false)
    })

    it('shows the path of the System, the Appearance and the Editor page and no action', async () => {
      const { tab } = createTab()

      await openSystemSettings(tab)
      expect(getSegments(tab)).toStrictEqual(['Settings'])
      expect(getCurrent(tab)).toBe(words.systemSettings)
      expect(getToolbar(tab).find('.settings-toolbar__actions').exists()).toBe(false)

      await goBack(tab)
      await openAppearanceSettings(tab)
      expect(getCurrent(tab)).toBe(words.appearanceSettings)

      await goBack(tab)
      await openEditorSettings(tab)
      expect(getCurrent(tab)).toBe('Editor')

      await goBack(tab)
      await openSearchSettings(tab)
      expect(getCurrent(tab)).toBe('Search')
    })

    it('shows the path of the Connections page with no action', async () => {
      const { tab } = createTab()

      await openPlugin(tab, 'Connections')

      expect(getSegments(tab)).toStrictEqual(['Settings'])
      expect(getCurrent(tab)).toBe(words.connections)
      expect(getToolbar(tab).find('.settings-toolbar__actions').exists()).toBe(false)
    })

    it('opens the Licenses page, shows Buy providers, activates a key and deactivates it', async () => {
      const sync: LicenseInfo = {
        id: 'numen.sync',
        title: 'Numen Sync',
        contributors: [
          {
            pluginId: 'md.numen.views.agent-chat',
            pluginName: 'Agent Chat',
            description: 'Multi-agent conversations and background tasks',
          },
        ],
        providers: [
          {
            provider: 'creem',
            providerName: 'Creem',
            productId: 'prod_creem_1',
            checkoutUrl: 'https://creem.io/checkout/numen-sync',
            paymentMethods: ['Cards', 'Apple Pay'],
          },
          {
            provider: 'dodo',
            providerName: 'Dodo Payments',
            productId: 'prod_dodo_1',
            checkoutUrl: 'https://checkout.dodopayments.com/buy/numen-sync',
            paymentMethods: ['UPI', 'PIX'],
          },
        ],
        status: 'inactive',
        isActive: false,
        hasKey: false,
        keyHint: '',
        activeProvider: '',
        instanceName: '',
        expiresUnixMs: 0,
        verifiedUnixMs: 0,
      }
      const list = ref<readonly LicenseInfo[]>([sync])
      const activate = vi.fn(async (licenseId: string, key: string) => {
        if (key === 'BAD-KEY') return asFailure('This license key was not recognized.')
        const activated: LicenseInfo = {
          ...sync,
          id: licenseId,
          status: 'active',
          isActive: true,
          hasKey: true,
          keyHint: '••••-814A',
          activeProvider: 'creem',
          instanceName: 'workstation',
        }
        list.value = [activated]
        return asValue(activated)
      })
      const deactivate = vi.fn(async () => {
        list.value = [sync]
        return asValue(sync)
      })
      const { tab } = createTab({}, [], SHELL, {
        licenses: {
          list,
          load: vi.fn(async () => undefined),
          watch: vi.fn(async () => undefined),
          activate,
          deactivate,
        },
      })

      await openPlugin(tab, 'Licenses')

      expect(getSegments(tab)).toStrictEqual(['Settings'])
      expect(getCurrent(tab)).toBe(words.licenses)
      expect(tab.text()).toContain('Numen Sync')
      expect(tab.text()).toContain('Agent Chat')
      expect(tab.text()).toContain('Multi-agent conversations and background tasks')

      // While input is empty, Buy dropdown button is shown and opens provider options.
      const buyBtn = tab.findAll('button').find((one) => one.text().includes('Buy'))
      expect(buyBtn).toBeDefined()
      await buyBtn!.trigger('click')
      await nextTick()
      const menuLinks = tab.findAll('a[role="menuitem"]')
      expect(menuLinks.map((one) => one.attributes('href'))).toStrictEqual([
        'https://creem.io/checkout/numen-sync',
        'https://checkout.dodopayments.com/buy/numen-sync',
      ])

      // Typing a key switches the contextual button from Buy to Activate.
      const keyInput = tab.get('input[aria-label="License key for Numen Sync"]')
      await keyInput.setValue('BAD-KEY')
      await nextTick()
      const activateBtn = tab.findAll('button').find((one) => one.text() === 'Activate')
      expect(activateBtn).toBeDefined()
      await activateBtn!.trigger('click')
      await nextTick()
      expect(tab.get('[role="alert"]').text()).toBe('This license key was not recognized.')

      await keyInput.setValue('GOOD-KEY-814A')
      await nextTick()
      await tab
        .findAll('button')
        .find((one) => one.text() === 'Activate')!
        .trigger('click')
      await nextTick()
      expect(activate).toHaveBeenCalledWith('numen.sync', 'GOOD-KEY-814A')
      expect(
        (tab.get('input[aria-label="License key for Numen Sync"]').element as HTMLInputElement)
          .value,
      ).toBe('••••-814A')

      const deactivateBtn = tab.findAll('button').find((one) => one.text() === 'Deactivate')
      expect(deactivateBtn).toBeDefined()
      await deactivateBtn!.trigger('click')
      await nextTick()
      expect(deactivate).toHaveBeenCalledWith('numen.sync')
      expect(tab.findAll('button').some((one) => one.text().includes('Buy'))).toBe(true)
    })

    it('holds the state and the enabled switch of a plugin on its page', async () => {
      const { tab } = createTab({}, [OCR])

      await openPlugin(tab, 'Page Reader')

      expect(getSegments(tab)).toStrictEqual(['Settings', 'Other'])
      expect(getCurrent(tab)).toBe('Page Reader')
      const actions = getToolbar(tab).get('.settings-toolbar__actions')
      expect(actions.text()).toContain('Running')
      expect(actions.get('[role="switch"]').attributes('aria-label')).toBe('Enabled Page Reader')
    })

    it('goes back to the list from the Settings segment', async () => {
      const { tab } = createTab({}, [OCR])
      await openPlugin(tab, 'Page Reader')

      await getPath(tab).findAll('button')[0]?.trigger('click')
      await nextTick()

      expect(isList(tab)).toBe(true)
    })

    it('returns to the list with the search kept from the category segment', async () => {
      const { tab, initialSearchQuery } = createTab({}, [OCR])
      initialSearchQuery.value = 'reader'
      await nextTick()
      await openPlugin(tab, 'Page Reader')

      await getPath(tab).findAll('button')[1]?.trigger('click')
      await nextTick()

      expect(isList(tab)).toBe(true)
      expect(initialSearchQuery.value).toBe('reader')
      expect((tab.get('input[type="search"]').element as HTMLInputElement).value).toBe('reader')
    })
  })
})
