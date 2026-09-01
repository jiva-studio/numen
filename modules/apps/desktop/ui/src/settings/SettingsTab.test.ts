/**
 * The settings tab drawn, in a document.
 *
 * What is asked here is that a row writes through the same value the command of
 * that name writes, and that a setting the window does not reach yet is drawn
 * where it belongs and says where it stands.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import SettingsTab from './SettingsTab.vue'
import type { Installation } from './kind'
import { WORDS as words } from './words'

/** An installation configured that way, and everything it was asked to change. */
const standing = (pinned = false) => {
  const done: string[] = []
  const syncing = ref(true)
  const hangs = ref(true)
  const installation: Installation = {
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
  }
  return { done, tab: mount(SettingsTab, { props: { held: { installation } } }) }
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
    expect(tab.findAll('optgroup').map((one) => one.attributes('label'))).toStrictEqual([
      words.shipped,
      words.owned,
    ])
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
    const switches = tab.findAll('[data-slot="switch"]')
    expect(switches).toHaveLength(2)
    await switches[0]!.trigger('click')
    await switches[1]!.trigger('click')
    expect(done).toStrictEqual(['hanging false', 'syncing false'])
  })

  it('draws a setting it does not write where it belongs, and says where it stands', () => {
    const { tab } = standing()
    const elsewhere = tab.findAll('.settings__elsewhere')
    expect(elsewhere).toHaveLength(5)
    expect(elsewhere[0]!.text()).toBe(words.inTheFile)
    expect(tab.text()).toContain(words.dayStarts)
    expect(tab.text()).toContain(words.agentUse)
  })
})
