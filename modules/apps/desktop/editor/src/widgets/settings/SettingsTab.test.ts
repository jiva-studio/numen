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

import type { Model, SettingEdit } from '../../entities/settings/configuration'
import { settingAt as at } from '../../entities/settings/store'
import SettingsTab from './SettingsTab.vue'
import type { Installation } from './useSettingsTab'
import { WORDS as words } from './words'

/** A model somebody typed into the file themselves, addressed and not named. */
const OWN = 'https://models.example/held/v3/rec/eslav_rec_mobile.onnx'

/** The models the vault offers, which is what the settings are grounded in. */
const MODELS: readonly Model[] = [
  {
    namedAt: ['agent', 'claude', 'model'],
    name: '',
    title: 'Whatever this machine answers with',
    shelf: '',
    byDefault: true,
    writes: [{ at: ['agent', 'claude', 'model'], value: '""' }],
    presence: 'nothing to fetch',
  },
  {
    namedAt: ['agent', 'claude', 'model'],
    name: 'opus',
    title: 'opus',
    shelf: 'By how large it is',
    byDefault: false,
    writes: [{ at: ['agent', 'claude', 'model'], value: '"opus"' }],
    presence: 'nothing to fetch',
  },
  {
    namedAt: ['indexing', 'recognition', 'recognise', 'name'],
    name: 'https://models.example/held/Tiny_rec.onnx',
    title: 'Tiny, small',
    shelf: '',
    byDefault: true,
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
    byDefault: false,
    writes: [{ at: ['indexing', 'recognition', 'recognise', 'name'], value: `"${OWN}"` }],
    presence: 'not fetched',
  },
]

/** An installation configured that way, and everything it was asked to change. */
const configured = (pinned = false, file: Record<string, unknown> = {}) => {
  const done: string[] = []
  const written: SettingEdit[] = []
  const syncing = ref(true)
  const hangs = ref(true)
  const installation: Installation = {
    setting: (path) => at(file, path),
    models: (path) => MODELS.filter((one) => one.namedAt.join('.') === path.join('.')),
    writes: (said) => void written.push(...said),
    file: ref('/numen.json'),
    opensFile: () => void done.push('opens the file'),
    themes: ref([
      { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
      { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
    ]),
    applied: ref('preset:numen'),
    mode: ref('system'),
    pinned: ref(pinned),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    chooses: (item) => void done.push(`chooses ${item}`),
    syncing: computed({
      get: () => syncing.value,
      set: (on) => {
        syncing.value = on
        done.push(`syncing ${on}`)
      },
    }),
    hangs: computed({
      get: () => hangs.value,
      set: (on) => {
        hangs.value = on
        done.push(`hanging ${on}`)
      },
    }),
    parts: ref(6),
    partsBounds: ref({ least: 2, most: 9 }),
    choosesParts: (count) => void done.push(`parts ${count}`),
    dayStarts: ref('04:00'),
    latestDayStarts: ref('12:00'),
    choosesDayStarts: (hour) => void done.push(`day starts ${hour}`),
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

type Tab = ReturnType<typeof configured>['tab']

/** The control on a row, which the name of that row announces. */
const control = (id: string) => `[aria-labelledby="${id}"]`

/** The choices one line offers, opened. They are drawn at the end of the document. */
const opens = async (tab: Tab, id: string) => {
  await tab.get(control(id)).trigger('click')
  await nextTick()
  await nextTick()
}

const offered = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__item .menu__text')).map(
    (one) => one.textContent?.trim() ?? '',
  )

const shelved = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__group-name')).map(
    (one) => one.textContent?.trim() ?? '',
  )

/** One choice taken off the open list, by what is written on it. */
const takes = async (words: string) => {
  const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item'))
  rows.find((one) => one.querySelector('.menu__text')?.textContent?.trim() === words)?.click()
  await nextTick()
}

describe('the settings tab', () => {
  it('draws every group, by the part of the application it governs', () => {
    const { tab } = configured()
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
    const { tab } = configured()
    expect(tab.get(control('settings-theme')).text()).toContain('numen')

    await opens(tab, 'settings-theme')
    expect(shelved()).toStrictEqual([words.shipped, words.owned])
  })

  it('writes a theme the way the command of that name writes it', async () => {
    const { tab, done } = configured()
    await opens(tab, 'settings-theme')
    await takes('sea')
    expect(done).toStrictEqual(['chooses mine:sea'])
  })

  it('says why the mode cannot be chosen while the theme worn pins it', () => {
    expect(configured(true).tab.text()).toContain(words.pinned)
    expect(configured().tab.text()).not.toContain(words.pinned)
  })

  it('turns the two switches the window keeps', async () => {
    const { tab, done } = configured()
    await tab.get(control('settings-hanging')).trigger('click')
    await tab.get(control('settings-syncing')).trigger('click')
    expect(done).toStrictEqual(['hanging false', 'syncing false'])
  })

  it('names the file the settings stand in, and opens it whole', async () => {
    const { tab, done } = configured()
    expect(tab.get('.settings__file').text()).toBe('/numen.json')

    await tab.get('.settings__where button').trigger('click')
    expect(done).toStrictEqual(['opens the file'])
  })

  it('carries no pencil, and no editor spliced under a row', () => {
    const { tab } = configured()
    expect(tab.findAll('.cm-editor')).toHaveLength(0)
    expect(tab.text()).not.toContain('JSON5')
  })

  it('draws the settings it reads out of the file, each under the group it is in', () => {
    const { tab } = configured()
    expect(tab.text()).toContain(words.indexingModel)
    expect(tab.text()).toContain(words.ocrModel)
    expect(tab.text()).toContain(words.transcribing)
    expect(tab.text()).toContain(words.agentUse)
  })

  it('holds the number of parts inside what the vault says it takes', () => {
    const { tab } = configured()
    const parts = tab.get(control('settings-parts'))

    expect(parts.attributes('aria-valuemin')).toBe('2')
    expect(parts.attributes('aria-valuemax')).toBe('9')
  })

  it('draws each of the two proofreadings beside the thing it puts right', () => {
    const { tab } = configured()
    const ocr = tab.get('section[aria-label="' + words.ocr + '"]')
    const heard = tab.get('section[aria-label="' + words.transcription + '"]')

    expect(ocr.text()).toContain(words.ocrProofread)
    expect(ocr.find(control('settings-ocr-proofread')).exists()).toBe(true)
    expect(heard.text()).toContain(words.transcriptProofread)
    expect(heard.find(control('settings-transcript-proofread')).exists()).toBe(true)
  })

  it('reads each of the two proofreadings out of its own path', () => {
    const { tab } = configured(false, {
      indexing: {
        recognition: { proofread: { with: 'careful' } },
        transcription: { proofread: { with: 'quick' } },
      },
    })
    expect(tab.get(control('settings-ocr-proofread')).text()).toContain('careful')
    expect(tab.get(control('settings-transcript-proofread')).text()).toContain('quick')
  })

  it('writes each of the two proofreadings into its own path', async () => {
    const { tab, written } = configured(false, {
      indexing: { proofreading: { profiles: { careful: {} } } },
    })
    await opens(tab, 'settings-transcript-proofread')
    await takes('careful')
    expect(written).toStrictEqual([
      { at: ['indexing', 'transcription', 'proofread', 'with'], value: '"careful"' },
    ])
  })

  it('turns whether each of the two is put right unasked', async () => {
    const { tab, written } = configured()
    await tab.get(control('settings-ocr-always')).trigger('click')
    await tab.get(control('settings-transcript-always')).trigger('click')
    expect(written).toStrictEqual([
      { at: ['indexing', 'recognition', 'proofread', 'automatically'], value: 'true' },
      { at: ['indexing', 'transcription', 'proofread', 'automatically'], value: 'true' },
    ])
  })

  it('opens a model on what the file names, and says which one is the default', async () => {
    const { tab } = configured(false, { agent: { claude: { model: 'opus' } } })
    expect(tab.get(control('settings-agent-model')).text()).toContain('opus')

    await opens(tab, 'settings-agent-model')
    expect(offered()).toContain(`Whatever this machine answers with — ${words.byDefault}`)
  })

  it('names a model by its own words, and addresses it underneath', async () => {
    const { tab } = configured()
    await opens(tab, 'settings-ocr')
    expect(offered()).toStrictEqual([
      `Tiny, small — ${words.byDefault}`,
      'eslav_rec_mobile.onnx',
    ])
    expect(document.body.querySelector('.menu__detail')?.textContent?.trim()).toBe(
      `${words.present} · https://models.example/held/Tiny_rec.onnx`,
    )
  })

  it('writes the readable name on the line, never the address it is fetched from', () => {
    const { tab } = configured(false, {
      indexing: { recognition: { recognise: { name: OWN } } },
    })
    const line = tab.get(control('settings-ocr'))

    expect(line.text()).toContain('eslav_rec_mobile.onnx')
    expect(line.text()).not.toContain('https://')
  })

  it('draws a value the presets do not name as the person’s own, and says nothing else', async () => {
    const own = 'https://models.example/mine/Other_rec.onnx'
    const { tab } = configured(false, {
      indexing: { recognition: { recognise: { name: own } } },
    })
    expect(tab.get(control('settings-ocr')).text()).toContain('Other_rec.onnx')
    expect(tab.text()).not.toMatch(/not found/i)

    await opens(tab, 'settings-ocr')
    expect(offered()[0]).toBe('Other_rec.onnx')
    expect(shelved()[0]).toBe(words.owned)
  })

  it('leaves a value the presets do not name alone while nothing is chosen', async () => {
    const own = 'https://models.example/mine/Other_rec.onnx'
    const { tab, written } = configured(false, {
      indexing: { recognition: { recognise: { name: own } } },
    })
    await opens(tab, 'settings-ocr')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()

    expect(written).toStrictEqual([])
    expect(tab.get(control('settings-ocr')).text()).toContain('Other_rec.onnx')
  })

  it('writes everything a model decides, not its name alone', async () => {
    const { tab, written } = configured()
    await opens(tab, 'settings-agent-model')
    await takes('opus')
    expect(written).toStrictEqual([{ at: ['agent', 'claude', 'model'], value: '"opus"' }])
  })

  it('writes a setting it read out of the file back where it stands', async () => {
    const { tab, written } = configured(false, { agent: { serve_tools: false } })
    await tab.get(control('settings-agent-tools')).trigger('click')
    expect(written).toStrictEqual([{ at: ['agent', 'serve_tools'], value: 'true' }])
  })

  it('offers the profiles the file holds, and naming none', async () => {
    const { tab } = configured(false, {
      indexing: { proofreading: { profiles: { careful: {}, quick: {} } } },
    })
    await opens(tab, 'settings-ocr-proofread')
    expect(offered()).toStrictEqual([words.proofreadingNone, 'careful', 'quick'])
  })

  it('opens on the hour the settings begin a day of review at', () => {
    const { tab } = configured()
    const hour = tab.get('[data-slot="time-field"]').element as HTMLInputElement
    expect(hour.value).toBe('04:00')
    expect(tab.text()).toContain(words.dayStarts)
  })

  it('writes the hour a day of review begins at', async () => {
    const { tab, done } = configured()
    await tab.get('[data-slot="time-field"]').setValue('06:30')
    expect(done).toStrictEqual(['day starts 06:30'])
  })

  it('names every row by what it is, without leaning on the row above', () => {
    const { tab } = configured()
    const names = tab.findAll('.settings__name').map((one) => one.text())
    expect(names).toContain(words.transcribeUnder)
    expect(names).not.toContain('Only under')
    expect(names).toContain(words.agentTools)
    expect(tab.text()).not.toContain('Tools on a port')
  })
})
