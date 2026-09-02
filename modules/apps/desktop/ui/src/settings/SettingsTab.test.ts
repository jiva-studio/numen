/**
 * The settings tab drawn, in a document.
 *
 * What is asked here is that a row writes through the same value the command of
 * that name writes, that a setting read out of the file is written back where
 * it stands, and that a model the file names and this build does not offer is
 * drawn as one that is not there.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { Model, Written } from '../core'
import { standing as at } from './configuring'
import SettingsTab from './SettingsTab.vue'
import type { Installation } from './kind'
import { WORDS as words } from './words'

/** The models the vault offers, which is what the settings are grounded in. */
const MODELS: readonly Model[] = [
  {
    namedAt: ['agent', 'claude', 'model'],
    name: '',
    title: 'Whatever this machine answers with',
    shelf: '',
    byDefault: true,
    writes: [{ at: ['agent', 'claude', 'model'], value: '""' }],
  },
  {
    namedAt: ['agent', 'claude', 'model'],
    name: 'opus',
    title: 'opus',
    shelf: 'By how large it is',
    byDefault: false,
    writes: [{ at: ['agent', 'claude', 'model'], value: '"opus"' }],
  },
]

/** An installation configured that way, and everything it was asked to change. */
const standing = (pinned = false, file: Record<string, unknown> = {}) => {
  const done: string[] = []
  const written: Written[] = []
  const syncing = ref(true)
  const hangs = ref(true)
  const installation: Installation = {
    setting: (path) => at(file, path),
    models: (path) => MODELS.filter((one) => one.namedAt.join('.') === path.join('.')),
    writes: (said) => void written.push(...said),
    file: () => '/numen.json',
    themes: () => [
      { name: 'preset:numen', title: 'numen', shipped: true, pinned: false },
      { name: 'mine:sea', title: 'sea', shipped: false, pinned: false },
    ],
    applied: () => 'preset:numen',
    mode: () => 'system',
    pinned: () => pinned,
    sizes: () => ({ interfaceScale: 1, textScale: 1 }),
    bounds: () => ({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    chooses: (item) => void done.push(`chooses ${item}`),
    syncing: () => syncing.value,
    choosesSyncing: (on) => {
      syncing.value = on
      done.push(`syncing ${on}`)
    },
    hangs: () => hangs.value,
    parts: () => 6,
    choosesHanging: (on) => {
      hangs.value = on
      done.push(`hanging ${on}`)
    },
    choosesParts: (count) => void done.push(`parts ${count}`),
    dayStarts: () => '04:00',
    choosesDayStarts: (hour) => void done.push(`day starts ${hour}`),
  }
  return { done, written, tab: mount(SettingsTab, { props: { held: { installation } } }) }
}

describe('the settings tab', () => {
  it('draws every group of the file, in the order the file keeps them', () => {
    const { tab } = standing()
    const headings = tab.findAll('.settings__heading').map((one) => one.text())
    expect(headings).toStrictEqual([
      words.window,
      words.naming,
      words.review,
      words.indexing,
      words.agent,
    ])
  })

  it('opens on the theme the settings name, off both shelves', () => {
    const { tab } = standing()
    const chosen = tab.get('select').element as HTMLSelectElement
    expect(chosen.value).toBe('preset:numen')
    expect(
      tab.get('#settings-theme').findAll('optgroup').map((one) => one.attributes('label')),
    ).toStrictEqual([words.shipped, words.owned])
  })

  it('writes a theme the way the command of that name writes it', async () => {
    const { tab, done } = standing()
    await tab.get('select').setValue('mine:sea')
    expect(done).toStrictEqual(['chooses mine:sea'])
  })

  it('says why the mode cannot be chosen while the theme worn pins it', () => {
    expect(standing(true).tab.text()).toContain(words.pinned)
    expect(standing().tab.text()).not.toContain(words.pinned)
  })

  it('turns the two switches the window keeps', async () => {
    const { tab, done } = standing()
    await tab.get('[aria-labelledby="settings-hanging"]').trigger('click')
    await tab.get('[aria-labelledby="settings-syncing"]').trigger('click')
    expect(done).toStrictEqual(['hanging false', 'syncing false'])
  })

  it('draws the settings it reads out of the file, each under the group it is in', () => {
    const { tab } = standing()
    expect(tab.text()).toContain(words.indexingModel)
    expect(tab.text()).toContain(words.ocr)
    expect(tab.text()).toContain(words.proofreading)
    expect(tab.text()).toContain(words.agentUse)
  })

  it('opens a model on what the file names, and says which one is the default', () => {
    const { tab } = standing(false, { agent: { claude: { model: 'opus' } } })
    const model = tab.get('#settings-agent-model').element as HTMLSelectElement
    expect(model.value).toBe('opus')
    expect(tab.text()).toContain(`Whatever this machine answers with — ${words.byDefault}`)
  })

  it('draws a model the file names and this build does not offer as one not there', () => {
    const { tab } = standing(false, { agent: { claude: { model: 'a-model-of-my-own' } } })
    const model = tab.get('#settings-agent-model').element as HTMLSelectElement
    expect(model.value).toBe('a-model-of-my-own')
    expect(tab.text()).toContain(`a-model-of-my-own — ${words.notFound}`)
  })

  it('writes everything a model decides, not its name alone', async () => {
    const { tab, written } = standing()
    await tab.get('#settings-agent-model').setValue('opus')
    expect(written).toStrictEqual([{ at: ['agent', 'claude', 'model'], value: '"opus"' }])
  })

  it('writes a setting it read out of the file back where it stands', async () => {
    const { tab, written } = standing(false, { agent: { serve_tools: false } })
    await tab.get('[aria-labelledby="settings-agent-tools"]').trigger('click')
    expect(written).toStrictEqual([{ at: ['agent', 'serve_tools'], value: 'true' }])
  })

  it('offers the profiles the file holds, and naming none', () => {
    const { tab } = standing(false, {
      indexing: { proofreading: { profiles: { careful: {}, quick: {} } } },
    })
    const profiles = tab.get('#settings-proofreading').findAll('option')
    expect(profiles.map((one) => one.text())).toStrictEqual([
      words.proofreadingNone,
      'careful',
      'quick',
    ])
  })

  it('opens on the hour the settings begin a day of review at', () => {
    const { tab } = standing()
    const hour = tab.get('[data-slot="time-field"]').element as HTMLInputElement
    expect(hour.value).toBe('04:00')
    expect(tab.text()).toContain(words.dayStarts)
  })

  it('writes the hour a day of review begins at', async () => {
    const { tab, done } = standing()
    await tab.get('[data-slot="time-field"]').setValue('06:30')
    expect(done).toStrictEqual(['day starts 06:30'])
  })
})
