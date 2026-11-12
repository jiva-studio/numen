/**
 * The page of the shell settings: every setting the core declares for the owner `app` is drawn by
 * the renderer the plugin pages use. Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import { Select, Slider, Switch } from '@numen/ui'
import SystemDetail from './SystemDetail.vue'
import SettingRow from './setting-row/SettingRow.vue'
import { createInstallation } from '@/pages/settings/testing/installation'
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

const THEMES = [
  { value: 'preset:numen', title: 'Numen', detail: '' },
  { value: 'plugin:md.example.themes/dusk', title: 'Dusk', detail: '' },
  { value: 'file:harbor', title: 'Harbor', detail: '' },
]

const DECLARATION: readonly PluginSettingInfo[] = [
  {
    key: 'appearance.theme',
    type: 'enum',
    title: 'Theme',
    description: 'The stylesheet the window is drawn with.',
    default: 'preset:numen',
    optionsFrom: 'themes',
    preview: true,
    activeOptions: THEMES,
    isShown: true,
    value: 'preset:numen',
  },
  {
    key: 'appearance.mode',
    type: 'enum',
    title: 'Light or dark',
    description: '',
    default: 'system',
    preview: true,
    options: [
      { value: 'system', title: 'System', detail: '' },
      { value: 'light', title: 'Light', detail: '' },
      { value: 'dark', title: 'Dark', detail: '' },
    ],
    activeOptions: [
      { value: 'system', title: 'System', detail: '' },
      { value: 'light', title: 'Light', detail: '' },
      { value: 'dark', title: 'Dark', detail: '' },
    ],
    isShown: true,
    value: 'dark',
  },
  {
    key: 'appearance.interfaceScale',
    type: 'range',
    title: 'Interface size',
    description: '',
    default: 1,
    min: 0.8,
    max: 2,
    step: 0.1,
    preview: true,
    isShown: true,
    value: 1.4,
  },
  {
    key: 'naming.syncTitleAndFilename',
    type: 'boolean',
    title: 'Sync title and filename',
    description: 'Renaming one also renames the other.',
    default: true,
    isShown: true,
    value: false,
  },
  {
    key: 'performance.profile',
    type: 'enum',
    title: 'Performance profile',
    description: '',
    default: 'balanced',
    options: [
      { value: 'eco', title: 'Eco', detail: 'Minimal power.' },
      { value: 'balanced', title: 'Balanced', detail: 'Standard workload.' },
    ],
    activeOptions: [
      { value: 'eco', title: 'Eco', detail: 'Minimal power.' },
      { value: 'balanced', title: 'Balanced', detail: 'Standard workload.' },
    ],
    isShown: true,
    value: 'balanced',
  },
]

const createPage = (
  section: 'system' | 'appearance',
  app: readonly PluginSettingInfo[] = DECLARATION,
) => {
  const writes: SettingEdit[] = []
  const installation = createInstallation({
    app: ref(app),
    write: (edits) => {
      writes.push(...edits)
    },
  })
  const wrapper = mount(SystemDetail, {
    props: {
      settings: installation.settings,
      manager: installation.plugin,
      connections: installation.connections,
      section,
    },
  })
  return { wrapper, writes }
}

const findRow = (wrapper: ReturnType<typeof createPage>['wrapper'], title: string) => {
  const row = wrapper.findAllComponents(SettingRow).find((one) => one.props('name') === title)
  if (!row) throw new Error(`no row is drawn for ${title}`)
  return row
}

describe('the page of the shell settings', () => {
  it('draws the appearance settings on Appearance and the rest on System, in the order declared', () => {
    const names = (section: 'system' | 'appearance') =>
      createPage(section)
        .wrapper.findAllComponents(SettingRow)
        .map((one) => one.props('name'))

    expect(names('appearance')).toStrictEqual(['Theme', 'Light or dark', 'Interface size'])
    expect(names('system')).toStrictEqual(['Sync title and filename', 'Performance profile'])
  })

  it('draws no row where the core declares no setting for the shell', () => {
    const { wrapper } = createPage('system', [])

    expect(wrapper.findAllComponents(SettingRow)).toHaveLength(0)
  })

  it('lists the themes of plugins and the ones the person owns among the themes to choose', () => {
    const { wrapper } = createPage('appearance')

    const choices = findRow(wrapper, 'Theme').findComponent(Select).props('choices')
    expect(choices.map((one) => one.id)).toStrictEqual([
      'preset:numen',
      'plugin:md.example.themes/dusk',
      'file:harbor',
    ])
    expect(choices.map((one) => one.text)).toStrictEqual(['Numen', 'Dusk', 'Harbor'])
  })

  it('shows the value the core resolved for each setting', () => {
    const { wrapper } = createPage('appearance')

    expect(findRow(wrapper, 'Theme').findComponent(Select).props('modelValue')).toBe('preset:numen')
    expect(findRow(wrapper, 'Light or dark').findComponent(Select).props('modelValue')).toBe('dark')
    expect(findRow(wrapper, 'Interface size').findComponent(Slider).props('modelValue')).toBe(1.4)
    expect(
      findRow(createPage('system').wrapper, 'Sync title and filename')
        .findComponent(Switch)
        .props('modelValue'),
    ).toBe(false)
  })

  it('bounds the scale by the bounds of its declaration', () => {
    const { wrapper } = createPage('appearance')

    expect(findRow(wrapper, 'Interface size').findComponent(Slider).props()).toMatchObject({
      min: 0.8,
      max: 2,
      step: 0.1,
    })
  })

  it('writes the theme chosen under the owner app', () => {
    const { wrapper, writes } = createPage('appearance')

    findRow(wrapper, 'Theme')
      .findComponent(Select)
      .vm.$emit('update:modelValue', 'plugin:md.example.themes/dusk')

    expect(writes).toStrictEqual([
      { at: ['app', 'appearance', 'theme'], value: '"plugin:md.example.themes/dusk"' },
    ])
  })

  it('writes the profile chosen under the owner app', () => {
    const { wrapper, writes } = createPage('system')

    findRow(wrapper, 'Performance profile')
      .findComponent(Select)
      .vm.$emit('update:modelValue', 'eco')

    expect(writes).toStrictEqual([{ at: ['app', 'performance', 'profile'], value: '"eco"' }])
  })

  it('writes the scale set under the owner app', () => {
    const { wrapper, writes } = createPage('appearance')

    findRow(wrapper, 'Interface size').findComponent(Slider).vm.$emit('commit', 1.2)

    expect(writes).toStrictEqual([{ at: ['app', 'appearance', 'interfaceScale'], value: '1.2' }])
  })

  it('writes the naming switch under the owner app', () => {
    const { wrapper, writes } = createPage('system')

    findRow(wrapper, 'Sync title and filename')
      .findComponent(Switch)
      .vm.$emit('update:modelValue', true)

    expect(writes).toStrictEqual([{ at: ['app', 'naming', 'syncTitleAndFilename'], value: 'true' }])
  })
})
