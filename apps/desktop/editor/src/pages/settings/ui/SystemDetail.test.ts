/**
 * The page of the shell settings: every setting the core declares for the owner `app` is drawn by
 * the renderer the plugin pages use. Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import SystemDetail from './SystemDetail.vue'
import SettingRow from './setting-row/SettingRow.vue'
import ConnectionRows from './plugins/connection-rows/ConnectionRows.vue'
import { ListField } from './list-field'
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
    type: 'string',
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

const SEARCH: readonly PluginSettingInfo[] = [
  RERANKING,
  {
    key: 'search.rerankWindow',
    type: 'number',
    title: 'Rerank window',
    description: '',
    default: 60,
    min: 1,
    max: 500,
    step: 1,
    isShown: true,
    value: 80,
  },
  {
    key: 'search.maxReranked',
    type: 'number',
    title: 'Most reranked passages',
    description: '',
    default: 300,
    min: 1,
    max: 2000,
    step: 1,
    isShown: true,
    value: 400,
  },
  {
    key: 'search.maxRerankedBytes',
    type: 'number',
    title: 'Longest reranked text',
    description: '',
    default: 4096,
    min: 256,
    max: 32768,
    step: 1,
    isShown: true,
    value: 2048,
  },
  {
    key: 'search.defaultLimit',
    type: 'number',
    title: 'Default results',
    description: '',
    default: 20,
    min: 1,
    max: 100,
    step: 1,
    isShown: true,
    value: 30,
  },
  {
    key: 'search.maxLimit',
    type: 'number',
    title: 'Most results',
    description: '',
    default: 100,
    min: 1,
    max: 1000,
    step: 1,
    isShown: true,
    value: 150,
  },
  {
    key: 'search.candidatesPerCondition',
    type: 'number',
    title: 'Passages per condition',
    description: '',
    default: 300,
    min: 1,
    max: 5000,
    step: 1,
    isShown: true,
    value: 500,
  },
  {
    key: 'search.maxCandidates',
    type: 'number',
    title: 'Most passages scored',
    description: '',
    default: 1000,
    min: 1,
    max: 10000,
    step: 1,
    isShown: true,
    value: 2000,
  },
  {
    key: 'search.maxPathPrefixes',
    type: 'number',
    title: 'Most folders',
    description: '',
    default: 8,
    min: 1,
    max: 64,
    step: 1,
    isShown: true,
    value: 12,
  },
  {
    key: 'search.mergeGapBytes',
    type: 'number',
    title: 'Merge gap',
    description: '',
    default: 200,
    min: 0,
    max: 4096,
    step: 1,
    isShown: true,
    value: 300,
  },
  {
    key: 'search.maxQueryBytes',
    type: 'number',
    title: 'Longest query',
    description: '',
    default: 16384,
    min: 256,
    max: 65536,
    step: 1,
    isShown: true,
    value: 8192,
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
  ...SEARCH,
]

const createPage = (
  section: 'system' | 'appearance' | 'editor' | 'search',
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
    const getRowNames = (section: 'system' | 'appearance' | 'editor' | 'search') =>
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

  it('draws no search setting on System, Appearance or Editor', () => {
    for (const section of ['system', 'appearance', 'editor'] as const) {
      const { wrapper } = createPage(section)
      const drawn = [
        ...wrapper.findAllComponents(SettingRow).map((one) => one.props('name')),
        ...wrapper.findAllComponents(ConnectionRows).map((one) => one.props('setting').title),
      ]
      for (const one of SEARCH) expect(drawn).not.toContain(one.title)
    }
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

describe('the Search page of the shell settings', () => {
  const SEARCH_NAMES = [
    'Rerank window',
    'Most reranked passages',
    'Longest reranked text',
    'Default results',
    'Most results',
    'Passages per condition',
    'Most passages scored',
    'Most folders',
    'Merge gap',
    'Longest query',
  ]

  it('draws the search settings in the order the core declares them, Reranking first', () => {
    const { wrapper } = createPage('search')

    const rerankingRows = wrapper.findAllComponents(ConnectionRows)
    expect(rerankingRows.map((one) => one.props('setting').title)).toStrictEqual(['Reranking'])
    expect(wrapper.findAllComponents(SettingRow).map((one) => one.props('name'))).toStrictEqual([
      'Reranking',
      ...SEARCH_NAMES,
    ])
  })

  it('draws each numeric search setting as a number field', () => {
    const { wrapper } = createPage('search')

    for (const name of SEARCH_NAMES) {
      expect(findRow(wrapper, name).findComponent(NumberField).exists()).toBe(true)
    }
  })

  it('shows the value the core resolved for each search setting', () => {
    const { wrapper } = createPage('search')

    expect(findRow(wrapper, 'Rerank window').findComponent(NumberField).props('modelValue')).toBe(
      80,
    )
    expect(findRow(wrapper, 'Default results').findComponent(NumberField).props('modelValue')).toBe(
      30,
    )
    expect(findRow(wrapper, 'Longest query').findComponent(NumberField).props('modelValue')).toBe(
      8192,
    )
  })

  it('bounds each search setting by the bounds of its declaration', () => {
    const { wrapper } = createPage('search')

    expect(findRow(wrapper, 'Merge gap').findComponent(NumberField).props()).toMatchObject({
      min: 0,
      max: 4096,
      step: 1,
    })
    expect(findRow(wrapper, 'Most results').findComponent(NumberField).props()).toMatchObject({
      min: 1,
      max: 1000,
      step: 1,
    })
  })

  it('writes the rows of Reranking chosen under app.search.reranking', () => {
    const { wrapper, writes } = createPage('search')

    wrapper
      .findComponent(ConnectionRows)
      .vm.$emit('change', [{ connection: 'c0000001', options: { model: 'rerank-small' } }])

    expect(writes.map((one) => [one.at, JSON.parse(one.value)])).toStrictEqual([
      [['app', 'search', 'reranking'], [{ connection: 'c0000001', model: 'rerank-small' }]],
    ])
  })

  it('writes the default results set under app.search', () => {
    const { wrapper, writes } = createPage('search')

    findRow(wrapper, 'Default results').findComponent(NumberField).vm.$emit('commit', 40)

    expect(writes).toStrictEqual([{ at: ['app', 'search', 'defaultLimit'], value: '40' }])
  })

  it('writes the merge gap and the longest query set under app.search', () => {
    const { wrapper, writes } = createPage('search')

    findRow(wrapper, 'Merge gap').findComponent(NumberField).vm.$emit('commit', 0)
    findRow(wrapper, 'Longest query').findComponent(NumberField).vm.$emit('commit', 1024)

    expect(writes).toStrictEqual([
      { at: ['app', 'search', 'mergeGapBytes'], value: '0' },
      { at: ['app', 'search', 'maxQueryBytes'], value: '1024' },
    ])
  })

  it('draws only the search settings, none of System, Appearance or Editor', () => {
    const { wrapper } = createPage('search')

    const names = wrapper.findAllComponents(SettingRow).map((one) => one.props('name'))
    for (const other of ['Theme', 'Font size', 'Sync title and filename', 'Performance profile']) {
      expect(names).not.toContain(other)
    }
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

  it('accepts an installed family typed into the font selector', () => {
    const { wrapper, writes } = createPage('editor')
    const select = findRow(wrapper, 'Font family').findComponent(Select)
    expect(select.props('canEnterCustom')).toBe(true)
    select.vm.$emit('update:modelValue', 'JetBrains Mono')
    expect(writes).toStrictEqual([
      { at: ['app', 'editor', 'fontFamily'], value: '"JetBrains Mono"' },
    ])
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
    const getRowNames = (section: 'system' | 'appearance') =>
      createPage(section)
        .wrapper.findAllComponents(SettingRow)
        .map((one) => one.props('name'))

    expect(getRowNames('system')).not.toContain('Font size')
    expect(getRowNames('system')).not.toContain('Soft wrap')
    expect(getRowNames('appearance')).not.toContain('Line height')
describe('a list setting of the shell', () => {
  const FOLDERS: PluginSettingInfo = {
    key: 'search.ignoredFolders',
    type: 'list',
    title: 'Ignored folders',
    description: 'Folders whose notes are not searched.',
    default: [],
    isShown: true,
    value: ['templates'],
  }

  it('draws a list setting of the app with the list control', () => {
    const { wrapper } = createPage('system', [FOLDERS])

    expect(
      findRow(wrapper, 'Ignored folders').findComponent(ListField).props('items'),
    ).toStrictEqual(['templates'])
  })

  it('writes the item added to a list setting of the app under the owner app', () => {
    const { wrapper, writes } = createPage('system', [FOLDERS])

    findRow(wrapper, 'Ignored folders').findComponent(ListField).vm.$emit('add', 'drafts')

    expect(
      writes.map((one) => ({ at: one.at, value: JSON.parse(one.value) as unknown })),
    ).toStrictEqual([{ at: ['app', 'search', 'ignoredFolders'], value: ['templates', 'drafts'] }])
  })
})
