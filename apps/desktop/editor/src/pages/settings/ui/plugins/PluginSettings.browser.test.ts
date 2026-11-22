import { afterEach, describe, expect, it } from 'vitest'
import { createApp, h, ref } from 'vue'
import '@numen/ui/styles.css'
import '@numen/plugin-manager'
import type { PluginSettingInfo } from '@/entities/settings'
import { createInstallation, createPluginInfo } from '@/pages/settings/testing/installation'
import PluginSettings from './PluginSettings.vue'

let host: HTMLElement | null = null

afterEach(() => {
  host?.remove()
  host = null
})

const toggle = (key: string, group?: string): PluginSettingInfo => ({
  key,
  type: 'boolean',
  title: key,
  description: '',
  default: false,
  ...(group ? { group } : {}),
})

function draw(): HTMLElement {
  const plugin = createPluginInfo('test.plugin', 'Engine', {
    settingGroups: [{ id: 'look', title: 'Look' }],
    settings: [toggle('plain-one'), toggle('plain-two'), toggle('a', 'look'), toggle('b', 'look')],
  })
  const installation = createInstallation({ plugins: ref([plugin]) })
  host = document.createElement('div')
  host.style.width = '720px'
  document.body.append(host)
  createApp({
    render: () =>
      h(PluginSettings, {
        settings: installation.settings,
        host: installation.plugin,
        plugin,
        marked: [],
        focused: null,
      }),
  }).mount(host)
  return host
}

const insets = (card: Element) => {
  const edge = card.getBoundingClientRect()
  const row = card.querySelector('.settings__row')!.getBoundingClientRect()
  const name = card.querySelector('.settings__name')!.getBoundingClientRect()
  const value = card.querySelector('.settings__value')!.getBoundingClientRect()
  return {
    name: name.left - edge.left,
    value: edge.right - value.right,
    dividerLeft: row.left - edge.left,
    dividerRight: edge.right - row.right,
    firstRowTop: row.top - edge.top,
  }
}

describe('the card of the ungrouped settings and the island of a group', () => {
  it('inset the text, the control and the dividers alike from the edges of the card', () => {
    const root = draw()
    const plain = root.querySelector('.settings-card')!
    const island = root.querySelector('section.island-card .island-card__body')!

    expect(insets(island)).toStrictEqual(insets(plain))
    expect(insets(plain).name).toBeGreaterThan(8)
  })
})
