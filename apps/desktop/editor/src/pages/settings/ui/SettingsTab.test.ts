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
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { Select, Switch } from '@numen/ui'

import type { Model, PluginInfo, PluginSettingInfo, SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/pages/settings/testing/settingAt'
import { createInstallation, createSettingsState } from '@/pages/settings/testing/installation'
import SettingsTab from './SettingsTab.vue'
import type { Installation } from '../types'
import { WORDS as words } from '../words'

/**
 * A model entered in the file by the user, identified by address and not by
 * name.
 */
const OWN = 'https://models.example/held/v3/rec/eslav_rec_mobile.onnx'

/** The models the vault offers, which is what the settings are grounded in. */
const MODELS: readonly Model[] = [
  {
    namedAt: ['agent', 'use'],
    name: 'claude',
    title: 'Claude Code',
    group: '',
    isDefault: true,
    writes: [{ at: ['agent', 'use'], value: '"claude"' }],
    presence: 'present',
  },
  {
    namedAt: ['agent', 'claude', 'model'],
    name: '',
    title: 'Default model',
    group: '',
    isDefault: true,
    writes: [{ at: ['agent', 'claude', 'model'], value: '""' }],
    presence: 'nothing to fetch',
  },
  {
    namedAt: ['agent', 'claude', 'model'],
    name: 'opus',
    title: 'opus',
    group: 'By how large it is',
    isDefault: false,
    writes: [{ at: ['agent', 'claude', 'model'], value: '"opus"' }],
    presence: 'nothing to fetch',
  },
  {
    namedAt: ['indexing', 'recognition', 'recognise', 'name'],
    name: 'https://models.example/held/Tiny_rec.onnx',
    title: 'Tiny, small',
    group: '',
    isDefault: true,
    writes: [
      { at: ['indexing', 'recognition', 'recognise', 'name'], value: '"held/Tiny_rec.onnx"' },
    ],
    presence: 'present',
  },
  // address in the place a name would be, which is all it has for one.
  {
    namedAt: ['indexing', 'recognition', 'recognise', 'name'],
    name: OWN,
    title: OWN,
    group: 'Named in the settings',
    isDefault: false,
    writes: [{ at: ['indexing', 'recognition', 'recognise', 'name'], value: `"${OWN}"` }],
    presence: 'not fetched',
  },
]

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

const createTab = (
  file: JsonObject = {},
  plugins: readonly PluginInfo[] = [],
  app: readonly PluginSettingInfo[] = SHELL,
) => {
  const content: SettingEdit[] = []
  const installation: Installation = createInstallation({
    getSetting: (path) => at(file, path),
    getModels: (path) => MODELS.filter((one) => one.namedAt.join('.') === path.join('.')),
    write: (response) => {
      content.push(...response)
    },
    plugins: ref(plugins),
    app: ref(app),
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

/** Every tab this file drew, put away between one test and the next. */
const mountedWrappers: { unmount: () => void }[] = []

afterEach(() => {
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
  await tab.find('.system-row').trigger('click')
  await nextTick()
}

describe('the settings tab', () => {
  it('draws the system settings row at the top of the list', () => {
    const { tab } = createTab()
    const rows = tab.findAll('.plugin-row')
    expect(rows.length).toBeGreaterThan(0)
    expect(rows[0]!.text()).toContain(words.systemSettings)
    expect(rows[0]!.text()).toContain(words.systemSettingsDetail)
  })

  it('draws the Connections row after system settings and plugin rows below them', () => {
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
    const rows = tab.findAll('.plugin-row')
    expect(rows.length).toBe(3)
    expect(rows[0]!.text()).toContain(words.systemSettings)
    expect(rows[1]!.text()).toContain('Connections')
    expect(rows[2]!.text()).toContain('Transcription Plugin')
  })

  it('opens on the theme the core resolved, in one list with the themes of plugins', async () => {
    const { tab } = createTab()
    await openSystemSettings(tab)
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
    await openSystemSettings(tab)
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

  it('names the file the settings stand in, and offers no tab to edit it', async () => {
    const { tab } = createTab()
    await openSystemSettings(tab)
    expect(tab.get('.system-detail__file').text()).toBe('/numen.json')
    expect(tab.find('.system-detail__where button').exists()).toBe(false)
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
    expect(tab.find('.system-detail__where').exists()).toBe(true)

    await tab.find('.system-detail__back').trigger('click')
    await nextTick()
    expect(tab.find('.system-detail').exists()).toBe(false)
    expect(tab.find('.system-row').exists()).toBe(true)
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

    // Click on the plugin row to select it
    const rows = tab.findAll('.plugin-row')
    expect(rows.length).toBe(3)
    await rows[2]!.trigger('click')
    await nextTick()

    // Full page plugin detail: system detail and list are gone
    expect(tab.find('.system-row').exists()).toBe(false)
    expect(tab.find('.plugin-detail').exists()).toBe(true)
    expect(tab.find('.plugin-detail__title').text()).toContain('Document OCR')

    // Click back button
    await tab.find('.plugin-detail__back').trigger('click')
    await nextTick()

    // Back to main settings page
    expect(tab.find('.plugin-detail').exists()).toBe(false)
    expect(tab.find('.system-row').exists()).toBe(true)
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
          operation: 'complete',
          max: 5,
          isShown: true,
        },
      ],
    }
    const { tab } = createTab({}, [proofreading])
    await tab.findAll('.plugin-row')[2]!.trigger('click')
    await nextTick()
    expect(tab.find('.connections-page').exists()).toBe(false)

    const open = tab.findAll('button').find((one) => one.text() === 'Open Connections')
    await open!.trigger('click')
    await nextTick()

    expect(tab.find('.connections-page').exists()).toBe(true)
    expect(tab.find('.plugin-detail').exists()).toBe(false)
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

    const searchInput = tab.get('.plugins-list__search-input')
    expect((searchInput.element as HTMLInputElement).value).toBe('feature:agent')

    await searchInput.setValue('')
    await nextTick()
    expect(initialSearchQuery.value).toBe('')

    const rows = tab.findAll('.plugin-row')
    await rows[2]!.trigger('click')
    await nextTick()
    expect(tab.find('.plugin-detail').exists()).toBe(true)

    await tab.find('.plugin-detail__back').trigger('click')
    await nextTick()

    const restoredInput = tab.get('.plugins-list__search-input')
    expect((restoredInput.element as HTMLInputElement).value).toBe('')
  })
})
