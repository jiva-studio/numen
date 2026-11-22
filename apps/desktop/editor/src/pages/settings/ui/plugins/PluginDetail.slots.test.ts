import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { mount } from '@vue/test-utils'
import type { JsonValue } from '@/shared/api/json'
import type { PluginInfo } from '@/entities/settings'
import { createInstallation, createPluginInfo } from '../../testing/installation'
import { getSettingAt as at } from '../../testing/settingAt'
import PluginDetail from './PluginDetail.vue'

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

afterEach(() => {
  vi.unstubAllGlobals()
})

const isSlots = (name: string | undefined): boolean =>
  name === 'PluginSlots' || name === 'PluginSlotRow'

const plugin = (): PluginInfo =>
  createPluginInfo('test.plugin', 'Test', {
    settings: [{ key: 'slots', type: 'slots', title: 'Slots', description: '', default: [] }],
  })

const groupsAt = (_read: number) => [
  {
    slot: 'titleBar.commands',
    title: 'Actions',
    contributors: [
      { key: 'a/command/one', label: 'one', pluginName: 'Test', group: 'x' },
      { key: 'a/command/two', label: 'two', pluginName: 'Test', group: 'y' },
    ],
  },
]

const stored = (): JsonValue => ({
  'titleBar.commands': [{ id: 'a/command/one' }, { id: '---' }, { id: 'a/command/two' }],
})

describe('the Slots tab of a plugin when the settings are read again with the same content', () => {
  it('create no island or row again and render no row again', async () => {
    const file = shallowRef<JsonValue>(stored())
    const refreshed = ref(0)
    const first = plugin()
    const installation = createInstallation({
      plugins: ref([first]),
      getSetting: (path) => at(file.value as never, path),
      getSlotGroups: () => groupsAt(refreshed.value),
    })
    let created = 0
    let rendered = 0
    const wrapper = mount(PluginDetail, {
      props: {
        settings: installation.settings,
        manager: installation.plugin,
        connections: installation.connections,
        plugin: first,
      },
      global: {
        mixins: [
          {
            mounted() {
              if (isSlots(this.$options.__name)) created += 1
            },
            updated() {
              if (isSlots(this.$options.__name)) rendered += 1
            },
          },
        ],
      },
    })
    await wrapper.get('[role="tab"][id$="slots"]').trigger('click')
    created = 0
    rendered = 0

    file.value = JSON.parse(JSON.stringify(file.value))
    refreshed.value += 1
    await wrapper.setProps({ plugin: structuredClone(first) })

    expect({ created, rendered }).toStrictEqual({ created: 0, rendered: 0 })
    expect(wrapper.findAll('li')).toHaveLength(3)
  })
})
