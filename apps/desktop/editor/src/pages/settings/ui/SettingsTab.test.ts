/**
 * The settings tab drawn, in a document.
 *
 * What is asked here is that a row writes through the same value the command of
 * that name writes, that a setting read out of the file is written back where
 * it stands, and that what the file holds is what is drawn as chosen.
 *
 * Every value here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { computed, nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { Model, PluginInfo, SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import SettingsTab from './SettingsTab.vue'
import type { Installation } from '../types'
import { WORDS as words } from '../words'

/** A model somebody typed into the file themselves, addressed and not named. */
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
    title: 'Whatever this machine answers with',
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

/** An installation configured that way, and everything it was asked to change. */
const createTab = (
  isPinned = false,
  file: Record<string, unknown> = {},
  plugins: readonly PluginInfo[] = [],
) => {
  const isDone: string[] = []
  const content: SettingEdit[] = []
  const isSyncing = ref(true)
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: (path) => MODELS.filter((one) => one.namedAt.join('.') === path.join('.')),
    write: (response) => {
      content.push(...response)
    },
    file: ref('/numen.json'),
    openFile: () => {
      isDone.push('opens the file')
    },
    themes: ref([
      { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
      { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
    ]),
    applied: ref('preset:numen'),
    mode: ref('system'),
    isPinned: ref(isPinned),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    choose: (item) => {
      isDone.push(`choose ${item}`)
    },
    isSyncing: computed({
      get: () => isSyncing.value,
      set: (on) => {
        isSyncing.value = on
        isDone.push(`syncing ${on}`)
      },
    }),
    writeSecret: vi.fn(),
    deleteSecret: vi.fn(),
    retryPlugin: vi.fn(),
    choosePath: vi.fn(async () => ''),
    plugins: ref(plugins),
  }
  const tab = mount(SettingsTab, {
    props: { state: { installation } },
    attachTo: document.body,
  })
  mountedWrappers.push(tab)
  return { recorded: isDone, written: content, recordedWrites: content, tab }
}

/** Every tab this file drew, put away between one test and the next. */
const mountedWrappers: { unmount: () => void }[] = []

afterEach(() => {
  while (mountedWrappers.length) mountedWrappers.pop()?.unmount()
})

type Tab = ReturnType<typeof createTab>['tab']

/** The control on a row, which the name of that row announces. */
const control = (id: string) => `[aria-labelledby="${id}"]`

/** The choices one line offers, opened. They are drawn at the end of the document. */
const openMenu = async (tab: Tab, id: string) => {
  await tab.get(control(id)).trigger('click')
  await nextTick()
  await nextTick()
}

const _getMenuItems = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__item .menu__text')).map(
    (one) => one.textContent?.trim() ?? '',
  )

const getMenuGroups = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__group-name')).map(
    (one) => one.textContent?.trim() ?? '',
  )

/** One choice taken off the open list, by what is written on it. */
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

  it('draws plugin rows below system settings when plugins are present', () => {
    const samplePlugin: PluginInfo = {
      id: 'test.transcription',
      name: 'Transcription Plugin',
      version: '1.0.0',
      description: 'Speech recognition engine',
      isActive: true,
      capabilities: [],
      secrets: [],
      storedSecrets: [],
      consent: 'none',
      hasFrontend: false,
      settings: [],
    }
    const { tab } = createTab(false, {}, [samplePlugin])
    const rows = tab.findAll('.plugin-row')
    expect(rows.length).toBe(2)
    expect(rows[0]!.text()).toContain(words.systemSettings)
    expect(rows[1]!.text()).toContain('Transcription Plugin')
  })

  it('opens on the theme the settings name, off both groups', async () => {
    const { tab } = createTab()
    await openSystemSettings(tab)
    expect(tab.get(control('settings-theme')).text()).toContain('numen')

    await openMenu(tab, 'settings-theme')
    expect(getMenuGroups()).toStrictEqual([words.shipped, words.owned])
  })

  it('writes a theme the way the command of that name writes it', async () => {
    const { tab, recorded } = createTab()
    await openSystemSettings(tab)
    await openMenu(tab, 'settings-theme')
    await chooseItem('sea')
    expect(recorded).toStrictEqual(['choose mine:sea'])
  })

  it('says why the mode cannot be chosen while the theme worn pins it', async () => {
    const tabPinned = createTab(true).tab
    await openSystemSettings(tabPinned)
    expect(tabPinned.text()).toContain(words.pinned)

    const tabUnpinned = createTab().tab
    await openSystemSettings(tabUnpinned)
    expect(tabUnpinned.text()).not.toContain(words.pinned)
  })

  it('turns the switch the window keeps for syncing', async () => {
    const { tab, recorded } = createTab()
    await openSystemSettings(tab)
    await tab.get(control('settings-syncing')).trigger('click')
    expect(recorded).toStrictEqual(['syncing false'])
  })

  it('names the file the settings stand in, and opens it whole', async () => {
    const { tab, recorded } = createTab()
    await openSystemSettings(tab)
    expect(tab.get('.system-detail__file').text()).toBe('/numen.json')

    await tab.get('.system-detail__where button').trigger('click')
    expect(recorded).toStrictEqual(['opens the file'])
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
      secrets: [],
      storedSecrets: [],
      consent: 'none',
      hasFrontend: false,
      settings: [],
    }
    const { tab } = createTab(false, {}, [samplePlugin])

    // Click on the plugin row to select it
    const rows = tab.findAll('.plugin-row')
    expect(rows.length).toBe(2)
    await rows[1]!.trigger('click')
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
})
