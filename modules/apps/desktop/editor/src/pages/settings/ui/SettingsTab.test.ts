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

import type { Model, SettingEdit } from '@/entities/settings'
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
  // What the vault answers with for a value standing in the settings: the
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
const createTab = (isPinned = false, file: Record<string, unknown> = {}) => {
  const isDone: string[] = []
  const content: SettingEdit[] = []
  const isSyncing = ref(true)
  const isNesting = ref(true)
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
    isNesting: computed({
      get: () => isNesting.value,
      set: (on) => {
        isNesting.value = on
        isDone.push(`nesting ${on}`)
      },
    }),
    headings: ref(6),
    headingsBounds: ref({ least: 2, most: 9 }),
    chooseHeadings: (count) => {
      isDone.push(`headings ${count}`)
    },
    writeSecret: vi.fn(),
    deleteSecret: vi.fn(),
    retryPlugin: vi.fn(),
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

const getMenuItems = (): readonly string[] =>
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

describe('the settings tab', () => {
  it('draws every group, by the part of the application it governs', () => {
    const { tab } = createTab()
    const headings = tab.findAll('.settings__heading').map((one) => one.text())
    expect(headings).toStrictEqual([
      words.window,
      words.naming,
      words.transcription,
      words.ocr,
      words.performance,
      words.indexing,
      words.agent,
      words.ai,
    ])
  })

  it('opens on the theme the settings name, off both groups', async () => {
    const { tab } = createTab()
    expect(tab.get(control('settings-theme')).text()).toContain('numen')

    await openMenu(tab, 'settings-theme')
    expect(getMenuGroups()).toStrictEqual([words.shipped, words.owned])
  })

  it('writes a theme the way the command of that name writes it', async () => {
    const { tab, recorded } = createTab()
    await openMenu(tab, 'settings-theme')
    await chooseItem('sea')
    expect(recorded).toStrictEqual(['choose mine:sea'])
  })

  it('says why the mode cannot be chosen while the theme worn pins it', () => {
    expect(createTab(true).tab.text()).toContain(words.pinned)
    expect(createTab().tab.text()).not.toContain(words.pinned)
  })

  it('turns the two switches the window keeps', async () => {
    const { tab, recorded } = createTab()
    await tab.get(control('settings-nesting')).trigger('click')
    await tab.get(control('settings-syncing')).trigger('click')
    expect(recorded).toStrictEqual(['nesting false', 'syncing false'])
  })

  it('names the file the settings stand in, and opens it whole', async () => {
    const { tab, recorded } = createTab()
    expect(tab.get('.settings__file').text()).toBe('/numen.json')

    await tab.get('.settings__where button').trigger('click')
    expect(recorded).toStrictEqual(['opens the file'])
  })

  it('carries no pencil, and no editor spliced under a row', () => {
    const { tab } = createTab()
    expect(tab.findAll('.cm-editor')).toHaveLength(0)
    expect(tab.text()).not.toContain('JSON5')
  })

  it('draws the settings it reads out of the file, each under the group it is in', () => {
    const { tab } = createTab()
    expect(tab.text()).toContain(words.indexingModel)
    expect(tab.text()).toContain(words.ocrModel)
    expect(tab.text()).toContain(words.transcribing)
    expect(tab.text()).toContain(words.agentUse)
  })

  it('holds the number of headings inside what the vault says it takes', () => {
    const { tab } = createTab()
    const headings = tab.get(control('settings-headings'))

    expect(headings.attributes('aria-valuemin')).toBe('2')
    expect(headings.attributes('aria-valuemax')).toBe('9')
  })

  it('draws each of the two proofreadings alongside the thing it puts right', () => {
    const { tab } = createTab()
    const ocr = tab.get('section[aria-label="' + words.ocr + '"]')
    const transcriptionSection = tab.get('section[aria-label="' + words.transcription + '"]')

    expect(ocr.text()).toContain(words.ocrProofread)
    expect(ocr.find(control('settings-ocr-proofread')).exists()).toBe(true)
    expect(transcriptionSection.text()).toContain(words.transcriptProofread)
    expect(transcriptionSection.find(control('settings-transcript-proofread')).exists()).toBe(true)
  })

  it('reads each of the two proofreadings out of its own path', () => {
    const { tab } = createTab(false, {
      indexing: {
        recognition: { proofread: { with: 'careful' } },
        transcription: { proofread: { with: 'quick' } },
      },
    })
    expect(tab.get(control('settings-ocr-proofread')).text()).toContain('careful')
    expect(tab.get(control('settings-transcript-proofread')).text()).toContain('quick')
  })

  it('writes each of the two proofreadings into its own path', async () => {
    const { tab, recordedWrites } = createTab(false, {
      indexing: { proofreading: { profiles: { careful: {} } } },
    })
    await openMenu(tab, 'settings-transcript-proofread')
    await chooseItem('careful')
    expect(recordedWrites).toStrictEqual([
      { at: ['indexing', 'transcription', 'proofread', 'with'], value: '"careful"' },
    ])
  })

  it('turns whether each of the two is put right unasked', async () => {
    const { tab, recordedWrites } = createTab()
    await tab.get(control('settings-ocr-always')).trigger('click')
    await tab.get(control('settings-transcript-always')).trigger('click')
    expect(recordedWrites).toStrictEqual([
      { at: ['indexing', 'recognition', 'proofread', 'automatically'], value: 'true' },
      { at: ['indexing', 'transcription', 'proofread', 'automatically'], value: 'true' },
    ])
  })

  it('opens a model on what the file names, and says which one is the default', async () => {
    const { tab } = createTab(false, { agent: { claude: { model: 'opus' } } })
    expect(tab.get(control('settings-agent-model')).text()).toContain('opus')

    await openMenu(tab, 'settings-agent-model')
    expect(getMenuItems()).toContain(`Whatever this machine answers with — ${words.byDefault}`)
  })

  it('names a model by its own words, and addresses it underneath', async () => {
    const { tab } = createTab()
    await openMenu(tab, 'settings-ocr')
    expect(getMenuItems()).toStrictEqual([
      `Tiny, small — ${words.byDefault}`,
      'eslav_rec_mobile.onnx',
    ])
    expect(document.body.querySelector('.menu__detail')?.textContent?.trim()).toBe(
      `${words.present} · https://models.example/held/Tiny_rec.onnx`,
    )
  })

  it('writes the readable name on the line, never the address it is fetched from', () => {
    const { tab } = createTab(false, {
      indexing: { recognition: { recognise: { name: OWN } } },
    })
    const line = tab.get(control('settings-ocr'))

    expect(line.text()).toContain('eslav_rec_mobile.onnx')
    expect(line.text()).not.toContain('https://')
  })

  it('draws a value the presets do not name as the person’s own, and says nothing else', async () => {
    const own = 'https://models.example/mine/Other_rec.onnx'
    const { tab } = createTab(false, {
      indexing: { recognition: { recognise: { name: own } } },
    })
    expect(tab.get(control('settings-ocr')).text()).toContain('Other_rec.onnx')
    expect(tab.text()).not.toMatch(/not found/i)

    await openMenu(tab, 'settings-ocr')
    expect(getMenuItems()[0]).toBe('Other_rec.onnx')
    expect(getMenuGroups()[0]).toBe(words.owned)
  })

  it('leaves a value the presets do not name alone while nothing is chosen', async () => {
    const own = 'https://models.example/mine/Other_rec.onnx'
    const { tab, recordedWrites } = createTab(false, {
      indexing: { recognition: { recognise: { name: own } } },
    })
    await openMenu(tab, 'settings-ocr')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()

    expect(recordedWrites).toStrictEqual([])
    expect(tab.get(control('settings-ocr')).text()).toContain('Other_rec.onnx')
  })

  it('writes everything a model decides, not its name alone', async () => {
    const { tab, recordedWrites } = createTab()
    await openMenu(tab, 'settings-agent-model')
    await chooseItem('opus')
    expect(recordedWrites).toStrictEqual([{ at: ['agent', 'claude', 'model'], value: '"opus"' }])
  })

  it('writes a setting it read out of the file back at its current position', async () => {
    const { tab, recordedWrites } = createTab(false, { agent: { serve_tools: false } })
    await tab.get(control('settings-agent-tools')).trigger('click')
    expect(recordedWrites).toStrictEqual([{ at: ['agent', 'serve_tools'], value: 'true' }])
  })

  it('offers the profiles the file holds, and naming none', async () => {
    const { tab } = createTab(false, {
      indexing: { proofreading: { profiles: { careful: {}, quick: {} } } },
    })
    await openMenu(tab, 'settings-ocr-proofread')
    expect(getMenuItems()).toStrictEqual([words.proofreadingNone, 'careful', 'quick'])
  })

  it('names every row by what it is, without leaning on the row above', () => {
    const { tab } = createTab()
    const names = tab.findAll('.settings__name').map((one) => one.text())
    expect(names).toContain(words.transcribeUnder)
    expect(names).not.toContain('Only under')
    expect(names).toContain(words.agentTools)
    expect(tab.text()).not.toContain('Tools on a port')
  })
})
