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
import ConnectionRows from './plugins/connection-rows/ConnectionRows.vue'
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

const FAMILIES = [
  { value: '', title: 'As in the theme', detail: '' },
  { value: 'sans', title: 'Sans', detail: '' },
  { value: 'serif', title: 'Serif', detail: '' },
  { value: 'mono', title: 'Mono', detail: '' },
]

const EDITOR: readonly PluginSettingInfo[] = [
  {
    key: 'editor.fontSize',
    type: 'range',
    title: 'Font size',
    description: 'In pixels.',
    default: 14,
    min: 10,
    max: 32,
    step: 1,
    isShown: true,
    value: 18,
  },
  {
    key: 'editor.fontFamily',
    type: 'enum',
    title: 'Font family',
    description: '',
    default: '',
    options: FAMILIES,
    activeOptions: FAMILIES,
    isShown: true,
    value: 'serif',
  },
  {
    key: 'editor.lineHeight',
    type: 'range',
    title: 'Line height',
    description: '',
    default: 1.6,
    min: 1.2,
    max: 2.4,
    step: 0.1,
    isShown: true,
    value: 1.8,
  },
  {
    key: 'editor.lineNumbers',
    type: 'boolean',
    title: 'Line numbers',
    description: '',
    default: false,
    isShown: true,
    value: true,
  },
  {
    key: 'editor.softWrap',
    type: 'boolean',
    title: 'Soft wrap',
    description: '',
    default: true,
    isShown: true,
    value: false,
  },
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
  ...EDITOR,
]

const createPage = (
  section: 'system' | 'appearance' | 'editor',
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
  it('draws the appearance settings on Appearance, the editor settings on Editor and the rest on System, in the order declared', () => {
    const getRowNames = (section: 'system' | 'appearance' | 'editor') =>
      createPage(section)
        .wrapper.findAllComponents(SettingRow)
        .map((one) => one.props('name'))

    expect(getRowNames('appearance')).toStrictEqual(['Theme', 'Light or dark', 'Interface size'])
    expect(getRowNames('system')).toStrictEqual(['Sync title and filename', 'Performance profile'])
    expect(getRowNames('editor')).toStrictEqual([
      'Font size',
      'Font family',
      'Line height',
      'Line numbers',
      'Soft wrap',
    ])
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

describe('the Reranking setting of the shell', () => {
  const RERANKING: PluginSettingInfo = {
    key: 'search.reranking',
    type: 'connections',
    title: 'Reranking',
    description: 'Services that order the results of a query, tried in order',
    default: undefined,
    operations: ['llm.rerank'],
    max: 5,
    isShown: true,
  }

  it('draws Reranking on System with the editor of connection rows and not on Appearance', () => {
    const declaration = [...DECLARATION, RERANKING]

    const rows = createPage('system', declaration).wrapper.findAllComponents(ConnectionRows)
    expect(rows.map((one) => one.props('setting').title)).toStrictEqual(['Reranking'])
    expect(
      createPage('appearance', declaration).wrapper.findAllComponents(ConnectionRows),
    ).toHaveLength(0)
  })

  it('writes the rows chosen under app.search.reranking', () => {
    const { wrapper, writes } = createPage('system', [...DECLARATION, RERANKING])

    wrapper
      .findComponent(ConnectionRows)
      .vm.$emit('change', [{ connection: 'c0000001', options: { model: 'rerank-small' } }])

    expect(writes.map((one) => [one.at, JSON.parse(one.value)])).toStrictEqual([
      [['app', 'search', 'reranking'], [{ connection: 'c0000001', model: 'rerank-small' }]],
    ])
  })
})

describe('the Editor page of the shell settings', () => {
  it('draws each editor setting with its own control', () => {
    const { wrapper } = createPage('editor')

    expect(findRow(wrapper, 'Font size').findComponent(Slider).exists()).toBe(true)
    expect(findRow(wrapper, 'Font family').findComponent(Select).exists()).toBe(true)
    expect(findRow(wrapper, 'Line height').findComponent(Slider).exists()).toBe(true)
    expect(findRow(wrapper, 'Line numbers').findComponent(Switch).exists()).toBe(true)
    expect(findRow(wrapper, 'Soft wrap').findComponent(Switch).exists()).toBe(true)
  })

  it('shows the value the core resolved for each editor setting', () => {
    const { wrapper } = createPage('editor')

    expect(findRow(wrapper, 'Font size').findComponent(Slider).props('modelValue')).toBe(18)
    expect(findRow(wrapper, 'Font family').findComponent(Select).props('modelValue')).toBe('serif')
    expect(findRow(wrapper, 'Line height').findComponent(Slider).props('modelValue')).toBe(1.8)
    expect(findRow(wrapper, 'Line numbers').findComponent(Switch).props('modelValue')).toBe(true)
    expect(findRow(wrapper, 'Soft wrap').findComponent(Switch).props('modelValue')).toBe(false)
  })

  it('bounds the font size from 10 to 32 pixels by 1 and the line height from 1.2 to 2.4 by 0.1', () => {
    const { wrapper } = createPage('editor')

    expect(findRow(wrapper, 'Font size').findComponent(Slider).props()).toMatchObject({
      min: 10,
      max: 32,
      step: 1,
    })
    expect(findRow(wrapper, 'Line height').findComponent(Slider).props()).toMatchObject({
      min: 1.2,
      max: 2.4,
      step: 0.1,
    })
  })

  it('offers the face of the theme, Sans, Serif and Mono to choose as the font family', () => {
    const { wrapper } = createPage('editor')

    const choices = findRow(wrapper, 'Font family').findComponent(Select).props('choices')
    expect(choices.map((one) => one.id)).toStrictEqual(['', 'sans', 'serif', 'mono'])
    expect(choices.map((one) => one.text)).toStrictEqual([
      'As in the theme',
      'Sans',
      'Serif',
      'Mono',
    ])
  })

  it('shows the face of the theme where the font family is empty', () => {
    const empty = EDITOR.map((one) =>
      one.key === 'editor.fontFamily' ? { ...one, value: '' } : one,
    )
    const { wrapper } = createPage('editor', empty)

    expect(findRow(wrapper, 'Font family').findComponent(Select).props('modelValue')).toBe('')
    expect(findRow(wrapper, 'Font family').text()).toContain('As in the theme')
  })

  it('writes the font size set under app.editor', () => {
    const { wrapper, writes } = createPage('editor')

    findRow(wrapper, 'Font size').findComponent(Slider).vm.$emit('commit', 22)

    expect(writes).toStrictEqual([{ at: ['app', 'editor', 'fontSize'], value: '22' }])
  })

  it('writes the font family chosen under app.editor', () => {
    const { wrapper, writes } = createPage('editor')

    findRow(wrapper, 'Font family').findComponent(Select).vm.$emit('update:modelValue', 'mono')

    expect(writes).toStrictEqual([{ at: ['app', 'editor', 'fontFamily'], value: '"mono"' }])
  })

  it('writes the empty family when the face of the theme is chosen', () => {
    const { wrapper, writes } = createPage('editor')

    findRow(wrapper, 'Font family').findComponent(Select).vm.$emit('update:modelValue', '')

    expect(writes).toStrictEqual([{ at: ['app', 'editor', 'fontFamily'], value: '""' }])
  })

  it('writes the line height set under app.editor', () => {
    const { wrapper, writes } = createPage('editor')

    findRow(wrapper, 'Line height').findComponent(Slider).vm.$emit('commit', 2.1)

    expect(writes).toStrictEqual([{ at: ['app', 'editor', 'lineHeight'], value: '2.1' }])
  })

  it('writes the line numbers and the soft wrap switched under app.editor', () => {
    const { wrapper, writes } = createPage('editor')

    findRow(wrapper, 'Line numbers').findComponent(Switch).vm.$emit('update:modelValue', false)
    findRow(wrapper, 'Soft wrap').findComponent(Switch).vm.$emit('update:modelValue', true)

    expect(writes).toStrictEqual([
      { at: ['app', 'editor', 'lineNumbers'], value: 'false' },
      { at: ['app', 'editor', 'softWrap'], value: 'true' },
    ])
  })

  it('keeps the editor settings off the System and the Appearance pages', () => {
    const names = (section: 'system' | 'appearance') =>
      createPage(section)
        .wrapper.findAllComponents(SettingRow)
        .map((one) => one.props('name'))

    expect(names('system')).not.toContain('Font size')
    expect(names('system')).not.toContain('Soft wrap')
    expect(names('appearance')).not.toContain('Line height')
  })
})
