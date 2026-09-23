/**
 * The settings tab drawn, in a document.
 *
 * What is asked here is that a row writes through the same value the command of
 * that name writes, that a setting read out of the file is written back where
 * it stands, and that what the file holds is what is drawn as chosen.
 *
 * Every value here is invented.
 */
import { afterEach, describe, expect, it } from 'vitest'
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
    namedAt: ['agent', 'claude', 'model'],
    name: '',
    title: 'Whatever this machine answers with',
    shelf: '',
    isDefault: true,
    writes: [{ at: ['agent', 'claude', 'model'], value: '""' }],
    presence: 'nothing to fetch',
  },
  {
    namedAt: ['agent', 'claude', 'model'],
    name: 'opus',
    title: 'opus',
    shelf: 'By how large it is',
    isDefault: false,
    writes: [{ at: ['agent', 'claude', 'model'], value: '"opus"' }],
    presence: 'nothing to fetch',
  },
  {
    namedAt: ['indexing', 'recognition', 'recognise', 'name'],
    name: 'https://models.example/held/Tiny_rec.onnx',
    title: 'Tiny, small',
    shelf: '',
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
    shelf: 'Named in the settings',
    isDefault: false,
    writes: [{ at: ['indexing', 'recognition', 'recognise', 'name'], value: `"${OWN}"` }],
    presence: 'not fetched',
  },
]

/** An installation configured that way, and everything it was asked to change. */
const createTab = (isPinned = false, file: Record<string, unknown> = {}) => {
  const done: string[] = []
  const written: SettingEdit[] = []
  const isSyncing = ref(true)
  const isHanging = ref(true)
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: (path) => MODELS.filter((one) => one.namedAt.join('.') === path.join('.')),
    write: (said) => void written.push(...said),
    file: ref('/numen.json'),
    openFile: () => void done.push('opens the file'),
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
    choose: (item) => void done.push(`choose ${item}`),
    isSyncing: computed({
      get: () => isSyncing.value,
      set: (on) => {
        isSyncing.value = on
        done.push(`syncing ${on}`)
      },
    }),
    isHanging: computed({
      get: () => isHanging.value,
      set: (on) => {
        isHanging.value = on
        done.push(`hanging ${on}`)
      },
    }),
    parts: ref(6),
    partsBounds: ref({ least: 2, most: 9 }),
    chooseParts: (count) => void done.push(`parts ${count}`),
    dayStarts: ref('04:00'),
    latestDayStarts: ref('12:00'),
    chooseDayStarts: (hour) => void done.push(`day starts ${hour}`),
  }
  const tab = mount(SettingsTab, {
    props: { state: { installation } },
    attachTo: document.body,
  })
  drawn.push(tab)
  return { done, written, tab }
}

/** Every tab this file drew, put away between one test and the next. */
const drawn: { unmount: () => void }[] = []

afterEach(() => {
  while (drawn.length) drawn.pop()?.unmount()
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
      words.review,
      words.transcription,
      words.ocr,
      words.indexing,
      words.agent,
    ])
  })

  it('opens on the theme the settings name, off both shelves', async () => {
    const { tab } = createTab()
    expect(tab.get(control('settings-theme')).text()).toContain('numen')

    await openMenu(tab, 'settings-theme')
    expect(getMenuGroups()).toStrictEqual([words.shipped, words.owned])
  })

  it('writes a theme the way the command of that name writes it', async () => {
    const { tab, done } = createTab()
    await openMenu(tab, 'settings-theme')
    await chooseItem('sea')
    expect(done).toStrictEqual(['choose mine:sea'])
  })

  it('says why the mode cannot be chosen while the theme worn pins it', () => {
    expect(createTab(true).tab.text()).toContain(words.pinned)
    expect(createTab().tab.text()).not.toContain(words.pinned)
  })

  it('turns the two switches the window keeps', async () => {
    const { tab, done } = createTab()
    await tab.get(control('settings-hanging')).trigger('click')
    await tab.get(control('settings-syncing')).trigger('click')
    expect(done).toStrictEqual(['hanging false', 'syncing false'])
  })

  it('names the file the settings stand in, and opens it whole', async () => {
    const { tab, done } = createTab()
    expect(tab.get('.settings__file').text()).toBe('/numen.json')

    await tab.get('.settings__where button').trigger('click')
    expect(done).toStrictEqual(['opens the file'])
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

  it('holds the number of parts inside what the vault says it takes', () => {
    const { tab } = createTab()
    const parts = tab.get(control('settings-parts'))

    expect(parts.attributes('aria-valuemin')).toBe('2')
    expect(parts.attributes('aria-valuemax')).toBe('9')
  })

  it('draws each of the two proofreadings beside the thing it puts right', () => {
    const { tab } = createTab()
    const ocr = tab.get('section[aria-label="' + words.ocr + '"]')
    const heard = tab.get('section[aria-label="' + words.transcription + '"]')

    expect(ocr.text()).toContain(words.ocrProofread)
    expect(ocr.find(control('settings-ocr-proofread')).exists()).toBe(true)
    expect(heard.text()).toContain(words.transcriptProofread)
    expect(heard.find(control('settings-transcript-proofread')).exists()).toBe(true)
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
    const { tab, written } = createTab(false, {
      indexing: { proofreading: { profiles: { careful: {} } } },
    })
    await openMenu(tab, 'settings-transcript-proofread')
    await chooseItem('careful')
    expect(written).toStrictEqual([
      { at: ['indexing', 'transcription', 'proofread', 'with'], value: '"careful"' },
    ])
  })

  it('turns whether each of the two is put right unasked', async () => {
    const { tab, written } = createTab()
    await tab.get(control('settings-ocr-always')).trigger('click')
    await tab.get(control('settings-transcript-always')).trigger('click')
    expect(written).toStrictEqual([
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
    const { tab, written } = createTab(false, {
      indexing: { recognition: { recognise: { name: own } } },
    })
    await openMenu(tab, 'settings-ocr')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()

    expect(written).toStrictEqual([])
    expect(tab.get(control('settings-ocr')).text()).toContain('Other_rec.onnx')
  })

  it('writes everything a model decides, not its name alone', async () => {
    const { tab, written } = createTab()
    await openMenu(tab, 'settings-agent-model')
    await chooseItem('opus')
    expect(written).toStrictEqual([{ at: ['agent', 'claude', 'model'], value: '"opus"' }])
  })

  it('writes a setting it read out of the file back where it stands', async () => {
    const { tab, written } = createTab(false, { agent: { serve_tools: false } })
    await tab.get(control('settings-agent-tools')).trigger('click')
    expect(written).toStrictEqual([{ at: ['agent', 'serve_tools'], value: 'true' }])
  })

  it('offers the profiles the file holds, and naming none', async () => {
    const { tab } = createTab(false, {
      indexing: { proofreading: { profiles: { careful: {}, quick: {} } } },
    })
    await openMenu(tab, 'settings-ocr-proofread')
    expect(getMenuItems()).toStrictEqual([words.proofreadingNone, 'careful', 'quick'])
  })

  it('opens on the hour the settings begin a day of review at', () => {
    const { tab } = createTab()
    const hour = tab.get('[data-slot="time-field"]').element as HTMLInputElement
    expect(hour.value).toBe('04:00')
    expect(tab.text()).toContain(words.dayStarts)
  })

  it('writes the hour a day of review begins at', async () => {
    const { tab, done } = createTab()
    await tab.get('[data-slot="time-field"]').setValue('06:30')
    expect(done).toStrictEqual(['day starts 06:30'])
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
