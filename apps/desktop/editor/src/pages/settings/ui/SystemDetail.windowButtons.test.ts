/**
 * The setting "Window buttons" on the page of the shell settings: the generic renderer draws it
 * from the declaration of the core. Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import { Select } from '@numen/ui'
import SystemDetail from './SystemDetail.vue'
import SettingRow from './setting-row/SettingRow.vue'
import { createInstallation } from '@/testing/installation'
import type { PluginSettingInfo, SettingEdit } from '@/entities/settings'

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

const OPTIONS = [
  { value: 'windows', title: 'Right', detail: 'Minimise, maximise and close at the right.' },
  { value: 'mac', title: 'Left', detail: 'Three circles at the left.' },
]

const BUTTONS: PluginSettingInfo = {
  key: 'window.buttons',
  type: 'enum',
  title: 'Window buttons',
  description: 'Where the window draws its buttons on Linux.',
  default: 'windows',
  options: OPTIONS,
  activeOptions: OPTIONS,
  isShown: true,
  value: 'mac',
}

const createPage = () => {
  const writes: SettingEdit[] = []
  const installation = createInstallation({
    app: ref([BUTTONS]),
    write: (edits) => {
      writes.push(...edits)
    },
  })
  const wrapper = mount(SystemDetail, { props: { state: { installation } } })
  return { wrapper, writes }
}

describe('the setting "Window buttons" on the page of the shell settings', () => {
  it('is drawn as a row with the title "Window buttons"', () => {
    const { wrapper } = createPage()

    expect(wrapper.findAllComponents(SettingRow).map((one) => one.props('name'))).toStrictEqual([
      'Window buttons',
    ])
  })

  it('offers the right and the left style and shows the one the file holds', () => {
    const { wrapper } = createPage()

    const select = wrapper.findComponent(Select)
    expect(select.props('choices').map((one) => one.id)).toStrictEqual(['windows', 'mac'])
    expect(select.props('modelValue')).toBe('mac')
  })

  it('writes the style chosen under the owner app', () => {
    const { wrapper, writes } = createPage()

    wrapper.findComponent(Select).vm.$emit('update:modelValue', 'windows')

    expect(writes).toStrictEqual([{ at: ['app', 'window', 'buttons'], value: '"windows"' }])
  })
})
