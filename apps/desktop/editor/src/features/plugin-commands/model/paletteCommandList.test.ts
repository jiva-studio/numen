import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Command } from '@/entities/command'
import { createPaletteCommandList } from './paletteCommandList'

const row = (id: string): Command => ({ id, text: id, group: 'window', isAvailable: () => true })

const declared = [
  { pluginId: 'org.example.tide', id: 'chart' },
  { pluginId: 'org.example.ledger', id: 'open' },
]

const setup = () =>
  createPaletteCommandList({
    getAppCommands: () => [row('app:note.save')],
    getSettingCommands: () => [row('setting:app:theme')],
    declaredCommands: {
      commands: ref(declared),
      paletteCommands: ref([row('org.example.tide:chart')]),
      slots: ref([]),
      whenContext: { getConfig: () => ({}), getContentTypes: () => ({}), whenReader: {} as never },
    } as never,
    getPluginEntries: () => [row('org.example.ledger:open')],
  })

describe('the commands of the palette besides the ones it owns', () => {
  it('are those of the features, the settings, the declarations and the plugins, in that order', () => {
    expect(
      setup()
        .listCommands()
        .map((one) => one.id),
    ).toStrictEqual([
      'app:note.save',
      'setting:app:theme',
      'org.example.tide:chart',
      'org.example.ledger:open',
    ])
  })

  it('that offer items are those of the features and the declarations', () => {
    expect(
      setup()
        .listAppCommands()
        .map((one) => one.id),
    ).toStrictEqual(['app:note.save', 'org.example.tide:chart'])
  })
})

describe('the commands of the plugins', () => {
  it('keep the order of the manifests whichever side lists them', () => {
    const list = createPaletteCommandList({
      getAppCommands: () => [],
      getSettingCommands: () => [],
      declaredCommands: {
        commands: ref(declared),
        paletteCommands: ref([row('org.example.ledger:open')]),
        slots: ref([]),
        whenContext: {
          getConfig: () => ({}),
          getContentTypes: () => ({}),
          whenReader: {} as never,
        },
      } as never,
      getPluginEntries: () => [row('org.example.tide:chart')],
    })

    expect(list.listCommands().map((one) => one.id)).toStrictEqual([
      'org.example.tide:chart',
      'org.example.ledger:open',
    ])
  })
})
