import type { Component } from 'vue'
import type { JsonObject } from '@/shared/api/json'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import PluginDetail from './PluginDetail.vue'
import SettingRow from '../setting-row/SettingRow.vue'
import ConnectionRows from './connection-rows/ConnectionRows.vue'
import { createInstallation, createPluginInfo } from '@/pages/settings/testing/installation'
import type { Installation } from '../../types'
import { getSettingAt as at } from '@/pages/settings/testing/settingAt'
import type { PluginInfo, PluginSettingInfo, SettingEdit } from '@/entities/settings'
import type { PluginHealthItem } from '@/entities/plugin'
import { WORDS as words } from '../../words'

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

const createDetail = (
  plugin: PluginInfo,
  file: JsonObject = {},
  platform?: string,
  attachTo?: HTMLElement,
  health: readonly PluginHealthItem[] = [],
) => {
  const recordedWrites: SettingEdit[] = []
  const retryPlugin = vi.fn()
  const runCommand = vi.fn()
  const installation: Installation = createInstallation({
    health: ref(health),
    runCommand,
    getSetting: (path) => at(file, path),
    write: (edits) => {
      recordedWrites.push(...edits)
    },
    retryPlugin,
    plugins: ref([plugin]),
    ...(platform ? { getPlatform: () => platform } : {}),
  })
  const wrapper = mount(PluginDetail, {
    props: {
      settings: installation.settings,
      manager: installation.plugin,
      connections: installation.connections,
      plugin,
    },
    ...(attachTo ? { attachTo } : {}),
  })
  return {
    wrapper,
    recordedWrites,
    retryPlugin,
    runCommand,
    installation,
  }
}

describe('PluginDetail', () => {
  it('renders plugin headline information', () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine', {
      version: '1.2.3',
      description: 'Engine description text',
    })
    const { wrapper } = createDetail(plugin)

    expect(wrapper.get('h2').text()).toBe('Super Engine')
    expect(wrapper.text()).toContain('v1.2.3')
    expect(wrapper.text()).toContain('Engine description text')
  })

  it('renders back button and clicking it emits back', async () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine')
    const { wrapper } = createDetail(plugin)

    await wrapper.get(`button[aria-label="${words.backToPlugins}"]`).trigger('click')
    expect(wrapper.emitted('back')).toHaveLength(1)
  })

  it('toggles master enabled switch and writes setting', async () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine')
    const { wrapper, recordedWrites } = createDetail(plugin)

    const masterSwitch = wrapper.getComponent(Switch)
    expect(masterSwitch.props('modelValue')).toBe(true)

    masterSwitch.vm.$emit('update:modelValue', false)

    expect(recordedWrites).toStrictEqual([{ at: ['test.plugin', 'enabled'], value: 'false' }])
  })

  it('reads master enabled switch state when disabled in settings', () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine')
    const { wrapper } = createDetail(plugin, {
      'test.plugin': { enabled: false },
    })

    const masterSwitch = wrapper.getComponent(Switch)
    expect(masterSwitch.props('modelValue')).toBe(false)
  })

  it('renders the failure card, shows its detail and calls retryPlugin', async () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine')
    const { wrapper, retryPlugin } = createDetail(
      plugin,
      {},
      undefined,
      undefined,
      [
        {
          pluginId: 'test.plugin',
          key: 'start',
          severity: 'error',
          text: 'Timed out while starting',
          detail: 'handshake timeout after 10s',
          action: { command: 'app:settings.open', values: { plugin: 'test.plugin' }, title: '' },
        },
      ],
    )

    const stoppedCard = wrapper.get('[role="group"][aria-label="Stopped"]')
    expect(stoppedCard.text()).toContain('Timed out while starting')
    expect(stoppedCard.text()).not.toContain('handshake timeout after 10s')

    const buttons = stoppedCard.findAll('button')
    await buttons.find((btn) => btn.text() === words.pluginShowDetails)!.trigger('click')
    expect(stoppedCard.get('code').text()).toBe('handshake timeout after 10s')

    await buttons.find((btn) => btn.text() === words.pluginRetry)!.trigger('click')
    expect(retryPlugin).toHaveBeenCalledExactlyOnceWith('test.plugin')
  })

  it('handles declarative settings controls: Switch, NumberField, Slider, Select, text input', async () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine', {
      settings: [
        {
          key: 'autoStart',
          type: 'boolean',
          title: 'Auto Start',
          description: 'Start automatically',
          default: false,
        },
        {
          key: 'retryCount',
          type: 'number',
          title: 'Retry Count',
          description: 'Number of retries',
          default: 3,
          min: 1,
          max: 10,
        },
        {
          key: 'confidence',
          type: 'range',
          title: 'Confidence Threshold',
          description: 'Range from 0 to 100',
          default: 80,
          min: 0,
          max: 100,
        },
        {
          key: 'logLevel',
          type: 'enum',
          title: 'Log Level',
          description: 'Severity level',
          default: 'info',
          options: [
            { value: 'debug', title: '', detail: '' },
            { value: 'info', title: '', detail: '' },
            { value: 'warn', title: '', detail: '' },
            { value: 'error', title: '', detail: '' },
          ],
        },
        {
          key: 'endpoint',
          type: 'string',
          title: 'API Endpoint',
          description: 'Server URL',
          default: 'https://api.example.com',
        },
      ],
    })
    const { wrapper, recordedWrites } = createDetail(plugin)

    // Switch
    const rows = wrapper.findAllComponents(SettingRow)
    const booleanRow = rows.find((r) => r.props('at') === 'test.plugin-autoStart')
    expect(booleanRow).toBeDefined()
    const switchComp = booleanRow!.findComponent(Switch)
    expect(switchComp.props('modelValue')).toBe(false)
    switchComp.vm.$emit('update:modelValue', true)
    expect(recordedWrites).toContainEqual({
      at: ['test.plugin', 'autoStart'],
      value: 'true',
    })

    // NumberField
    const numberRow = rows.find((r) => r.props('at') === 'test.plugin-retryCount')
    expect(numberRow).toBeDefined()
    const numberComp = numberRow!.findComponent(NumberField)
    expect(numberComp.props('modelValue')).toBe(3)
    numberComp.vm.$emit('commit', 5)
    expect(recordedWrites).toContainEqual({
      at: ['test.plugin', 'retryCount'],
      value: '5',
    })

    // Slider
    const rangeRow = rows.find((r) => r.props('at') === 'test.plugin-confidence')
    expect(rangeRow).toBeDefined()
    const sliderComp = rangeRow!.findComponent(Slider)
    expect(sliderComp.props('modelValue')).toBe(80)
    expect(rangeRow!.find('.settings__slider-value').text()).toBe('80')
    sliderComp.vm.$emit('commit', 90)
    expect(recordedWrites).toContainEqual({
      at: ['test.plugin', 'confidence'],
      value: '90',
    })

    // Select
    const enumRow = rows.find((r) => r.props('at') === 'test.plugin-logLevel')
    expect(enumRow).toBeDefined()
    const selectComp = enumRow!.findComponent(Select)
    expect(selectComp.props('modelValue')).toBe('info')
    selectComp.vm.$emit('update:modelValue', 'debug')
    expect(recordedWrites).toContainEqual({
      at: ['test.plugin', 'logLevel'],
      value: '"debug"',
    })

    // Text Input
    const textRow = rows.find((r) => r.props('at') === 'test.plugin-endpoint')
    expect(textRow).toBeDefined()
    const textInput = textRow!.find('.settings__text-input')
    expect((textInput.element as HTMLInputElement).value).toBe('https://api.example.com')
    await textInput.setValue('https://custom.example.com')
    await textInput.trigger('change')
    expect(recordedWrites).toContainEqual({
      at: ['test.plugin', 'endpoint'],
      value: '"https://custom.example.com"',
    })
  })

  it('keeps the text being typed when the settings are drawn again before it is written', async () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine', {
      settings: [
        { key: 'endpoint', type: 'string', title: 'Endpoint', description: '', default: '' },
      ],
    })
    const { wrapper, recordedWrites } = createDetail(plugin)
    const input = wrapper.find<HTMLInputElement>('.settings__text-input')
    input.element.value = 'http://127.0.0.1:9/v1'
    await input.trigger('input')

    await wrapper.setProps({ plugin: { ...plugin } })
    expect(wrapper.find<HTMLInputElement>('.settings__text-input').element.value).toBe(
      'http://127.0.0.1:9/v1',
    )

    await wrapper.find('.settings__text-input').trigger('change')
    expect(recordedWrites).toStrictEqual([
      { at: ['test.plugin', 'endpoint'], value: '"http://127.0.0.1:9/v1"' },
    ])
  })

  it('says there is nothing to set up for a plugin without settings, and keeps its switch', () => {
    const plugin = createPluginInfo('test.empty', 'Empty Plugin')
    const { wrapper } = createDetail(plugin)

    expect(wrapper.get('h2').text()).toBe('Empty Plugin')
    expect(wrapper.findComponent(Switch).exists()).toBe(true)
    expect(wrapper.get('[role="tabpanel"]').text()).toBe(words.noSettings)
  })

  it('puts the back button first, above the title', () => {
    const { wrapper } = createDetail(createPluginInfo('test.plugin', 'Super Engine'))
    const buttons = wrapper.findAll('button')
    expect(buttons[0]?.attributes('aria-label')).toBe(words.backToPlugins)
  })

  it('opens on Settings and draws the README on About without its title and comments', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: true,
        headers: { get: () => 'text/markdown' },
        text: async () =>
          '# Super Engine\n\nShort summary line.\n\n<!-- more -->\n\nDetailed documentation body.',
      })),
    )
    const { wrapper } = createDetail(
      createPluginInfo('a', 'Super Engine', { description: 'Fallback' }),
    )
    await flushPromises()

    expect(wrapper.get('[role="tab"][aria-selected="true"]').text()).toBe('Settings')
    await wrapper
      .findAll('[role="tab"]')
      .find((one) => one.text() === 'About')!
      .trigger('click')

    const panel = wrapper.get('[role="tabpanel"]')
    expect(panel.text()).toContain('Short summary line.')
    expect(panel.text()).toContain('Detailed documentation body.')
    expect(panel.text()).not.toContain('Super Engine')
    expect(panel.html()).not.toContain('more')
  })

  it('offers no Show details when the failure detail is empty', () => {
    const start: PluginHealthItem = {
      pluginId: 'a',
      key: 'start',
      severity: 'error',
      text: 'Failed to start',
      detail: '',
    }
    const { wrapper } = createDetail(createPluginInfo('a', 'A'), {}, undefined, undefined, [start])
    const card = wrapper.get('[role="group"][aria-label="Stopped"]')
    expect(card.findAll('button').map((one) => one.text())).toStrictEqual([words.pluginRetry])
  })
})

describe('PluginDetail with a health item', () => {
  const ATLAS = createPluginInfo('org.example.atlas', 'Atlas', {
    settings: [
      {
        key: 'source',
        type: 'enum',
        title: 'Source',
        description: 'Where the atlas reads its maps',
        default: 'local',
        options: [
          { value: 'local', title: 'Local maps', detail: '' },
          { value: 'connection', title: 'Connection', detail: '' },
        ],
      },
      { key: 'zoom', type: 'number', title: 'Zoom', description: '', default: 3 },
    ],
  })
  const CONNECTION: PluginHealthItem = {
    pluginId: 'org.example.atlas',
    key: 'connection',
    severity: 'error',
    text: 'Choose a connection',
    detail: 'The atlas has no connection to ask.',
    action: {
      command: 'app:settings.open',
      values: { plugin: 'org.example.atlas', setting: 'source' },
      title: '',
    },
  }

  it('shows the item in a bar and runs its fix with its values', async () => {
    const { wrapper, runCommand } = createDetail(ATLAS, {}, undefined, undefined, [CONNECTION])
    const bar = wrapper.get('[role="alert"]')

    expect(bar.text()).toContain('Choose a connection')
    expect(bar.text()).toContain('The atlas has no connection to ask.')
    await bar.find('button').trigger('click')

    expect(runCommand).toHaveBeenCalledExactlyOnceWith('app:settings.open', {
      plugin: 'org.example.atlas',
      setting: 'source',
    })
  })

  it('marks the setting the fix names and no other', () => {
    const { wrapper } = createDetail(ATLAS, {}, undefined, undefined, [CONNECTION])

    const marked = wrapper.findAllComponents(SettingRow).filter((one) => one.props('isMarked'))
    expect(marked.map((one) => one.props('name'))).toStrictEqual(['Source'])
  })
})

describe('PluginDetail with settings that follow one another', () => {
  const option = (value: string, title = '', detail = '') => ({ value, title, detail })
  const LINKED = createPluginInfo('test.linked', 'Linked', {
    settings: [
      {
        key: 'model',
        type: 'enum',
        title: 'Model',
        description: '',
        default: 'large',
        options: [option('large', 'Large'), option('small', 'Small')],
      },
      {
        key: 'language',
        type: 'enum',
        title: 'Language',
        description: 'Spoken language',
        default: undefined,
        optionsBy: {
          setting: 'model',
          options: {
            large: [
              option('auto', 'Detected by the model', 'Among 3 languages'),
              option('en', 'English'),
            ],
            small: [option('en', 'English')],
          },
        },
      },
      {
        key: 'provider',
        type: 'enum',
        title: 'Provider',
        description: '',
        default: 'local',
        options: [option('local'), option('remote')],
      },
      {
        key: 'url',
        type: 'string',
        title: 'URL',
        description: '',
        default: '',
        when: { setting: 'provider', is: ['remote'] },
      },
      {
        key: 'dictionary',
        type: 'file',
        title: 'Dictionary',
        description: 'One character per line',
        default: '',
        accept: ['.txt'],
      },
    ],
  })
  /** The plugin as the core answers it: the options, visibility and value it resolved, by key. */
  const resolveBy = (
    plugin: PluginInfo,
    resolved: Record<string, Partial<PluginSettingInfo>>,
  ): PluginInfo => ({
    ...plugin,
    settings: plugin.settings.map((one) => ({ ...one, ...resolved[one.key] })),
  })
  const findRow = (wrapper: ReturnType<typeof mount>, key: string) =>
    wrapper.findAllComponents(SettingRow).find((r) => r.props('at') === `test.linked-${key}`)

  it('lists the languages of the current model with detail subtitles and no origin text', () => {
    const { wrapper } = createDetail(
      resolveBy(LINKED, {
        language: {
          activeOptions: [
            option('auto', 'Detected by the model', 'Among 3 languages'),
            option('en', 'English'),
          ],
          isShown: true,
          value: 'auto',
        },
        url: { isShown: false },
      }),
    )
    const row = findRow(wrapper, 'language')!
    expect(row.findComponent(Select).props('choices')).toEqual([
      { id: 'auto', text: 'Detected by the model', detail: 'Among 3 languages' },
      { id: 'en', text: 'English' },
    ])
    expect(row.props('detail')).toBe('Spoken language')
    expect(row.find('.settings__origin').exists()).toBe(false)
  })

  it('shows the one language of a model as fixed', () => {
    const small = createDetail(
      resolveBy(LINKED, {
        language: { activeOptions: [option('en', 'English')], isShown: true, value: 'en' },
        url: { isShown: false },
      }),
      { 'test.linked': { model: 'small' } },
    )
    const row = findRow(small.wrapper, 'language')!
    expect(row.findComponent(Select).exists()).toBe(false)
    expect(row.find('.settings__fixed').text()).toBe('English')
  })

  it("falls back to the active option when a string setting holds another provider's value", () => {
    const embeddingsPlugin = createPluginInfo('md.numen.engines.embeddings', 'Embeddings', {
      settings: [
        {
          key: 'provider',
          type: 'enum',
          title: 'Provider',
          description: '',
          default: 'local',
          options: [option('local', 'Local'), option('openrouter', 'OpenRouter')],
        },
        {
          key: 'model',
          type: 'string',
          title: 'Model',
          description: '',
          default: 'Xenova/bge-m3',
          optionsBy: {
            setting: 'provider',
            options: {
              local: [
                option('Xenova/bge-m3', 'BGE-M3'),
                option('Xenova/multilingual-e5-small', 'E5 Small'),
              ],
              openrouter: [option('openai/text-embedding-3-small', 'OpenAI Small')],
            },
          },
        },
      ],
    })
    const { wrapper } = createDetail(
      resolveBy(embeddingsPlugin, {
        model: {
          activeOptions: [
            option('Xenova/bge-m3', 'BGE-M3'),
            option('Xenova/multilingual-e5-small', 'E5 Small'),
          ],
          isShown: true,
          value: 'Xenova/bge-m3',
        },
      }),
      {
        'md.numen.engines.embeddings': {
          provider: 'local',
          model: 'openai/text-embedding-3-small',
        },
      },
    )
    const row = wrapper
      .findAllComponents(SettingRow)
      .find((r) => r.props('at') === 'md.numen.engines.embeddings-model')!
    expect(row.findComponent(Select).props('modelValue')).toBe('Xenova/bge-m3')
  })

  it('shows provider settings only while the provider is selected', () => {
    const local = createDetail(resolveBy(LINKED, { url: { isShown: false } }))
    expect(findRow(local.wrapper, 'url')).toBeUndefined()

    const remote = createDetail(
      resolveBy(LINKED, { provider: { value: 'remote' }, url: { isShown: true } }),
      { 'test.linked': { provider: 'remote' } },
    )
    expect(findRow(remote.wrapper, 'url')).toBeDefined()
    const rowIds = remote.wrapper.findAllComponents(SettingRow).map((r) => r.props('at'))
    const providerIndex = rowIds.indexOf('test.linked-provider')
    const urlIndex = rowIds.indexOf('test.linked-url')
    expect(providerIndex).toBeGreaterThanOrEqual(0)
    expect(urlIndex).toBeGreaterThan(providerIndex)
  })

  it('chooses a path with the picker, says why it was not taken, and clears it', async () => {
    const { wrapper, installation, recordedWrites } = createDetail(LINKED, {
      'test.linked': { dictionary: '/home/a/dicts/ru/letters.txt' },
    })
    const choosePath = vi.fn(async () => 'letters.onnx: wrong type of file, expected .txt')
    installation.plugin.choosePath = choosePath
    const row = findRow(wrapper, 'dictionary')!
    const label = row.find('[data-testid="setting-path"]')
    expect(label.text()).toBe('…/ru/letters.txt')
    expect(label.attributes('title')).toBe('/home/a/dicts/ru/letters.txt')

    const [choose, clear] = row.find('.settings__value').findAll('button')
    await choose!.trigger('click')
    await flushPromises()
    expect(choosePath).toHaveBeenCalledWith('test.linked', 'dictionary')
    expect(row.find('[role="alert"]').text()).toContain('wrong type of file')

    await clear!.trigger('click')
    expect(recordedWrites).toContainEqual({
      at: ['test.linked', 'dictionary'],
      value: '""',
    })
  })

  it('hides the path label when no path is chosen and lists each path of a multiple setting', async () => {
    const emptyDetail = createDetail(LINKED)
    const emptyRow = findRow(emptyDetail.wrapper, 'dictionary')!
    expect(emptyRow.find('[data-testid="setting-path"]').exists()).toBe(false)

    const acpPlugin: PluginInfo = {
      ...createPluginInfo('test.acp', 'ACP Agents'),
      settings: [
        {
          key: 'programs',
          type: 'file',
          multiple: true,
          title: 'Programs',
          description: '',
          default: '/usr/local/bin/gemini\n/usr/local/bin/opencode',
        },
      ],
    }
    const acpDetail = createDetail(acpPlugin)
    const acpRow = acpDetail.wrapper.findAllComponents(SettingRow)[0]!
    const autoFound = acpRow.findAll('[data-testid="setting-path"]')
    expect(autoFound).toHaveLength(2)
    expect(autoFound[0]!.text()).toBe('/usr/local/bin/gemini')
    expect(autoFound[1]!.text()).toBe('/usr/local/bin/opencode')
    const acpClearButtons = acpRow
      .findAll('button')
      .filter((btn) => btn.attributes('aria-label') === words.pathClear)
    expect(acpClearButtons).toHaveLength(2)
    await acpClearButtons[0]!.trigger('click')
    expect(acpDetail.recordedWrites).toContainEqual({
      at: ['test.acp', 'programs'],
      value: '"/usr/local/bin/opencode"',
    })
    const addButton = acpRow.findAll('button').find((btn) => btn.text() === words.pathAdd)
    expect(addButton).toBeDefined()
  })

  const createPathPlugin = (multiple: boolean): PluginInfo => ({
    ...createPluginInfo('test.paths', 'Paths'),
    settings: [
      {
        key: 'executablePath',
        type: 'file',
        ...(multiple ? { multiple: true } : {}),
        title: 'Executable Paths',
        description: '',
        default: '',
      },
    ],
  })

  it('draws a file setting that is not multiple as one path, whatever its title and key', () => {
    const { wrapper } = createDetail(createPathPlugin(false), {
      'test.paths': { executablePath: '/opt/one\n/opt/two' },
    })
    const row = wrapper.findAllComponents(SettingRow)[0]!

    expect(row.findAll('[data-testid="setting-path"]')).toHaveLength(1)
    expect(row.findAll('button').map((btn) => btn.text())).not.toContain(words.pathAdd)
  })

  /** Chooses /opt/two in the picker, which the host stores in place of the setting's value. */
  const chooseSecondPath = async (multiple: boolean) => {
    const stored = { executablePath: '/opt/one' }
    const detail = createDetail(createPathPlugin(multiple), { 'test.paths': stored })
    detail.installation.plugin.choosePath = vi.fn(async () => {
      stored.executablePath = '/opt/two'
      return ''
    })
    const row = detail.wrapper.findAllComponents(SettingRow)[0]!
    const label = multiple ? words.pathAdd : words.pathChoose
    await row
      .findAll('button')
      .find((btn) => btn.text() === label)!
      .trigger('click')
    await flushPromises()
    return detail.recordedWrites
  }

  it('adds the chosen path to the paths a multiple setting holds', async () => {
    expect(await chooseSecondPath(true)).toContainEqual({
      at: ['test.paths', 'executablePath'],
      value: '"/opt/one\\n/opt/two"',
    })
  })

  it('replaces the path of a setting that is not multiple', async () => {
    expect(await chooseSecondPath(false)).toStrictEqual([])
  })
})

describe('the settings views of a plugin', () => {
  const View = defineComponent({
    props: { pluginId: { type: String, required: true }, values: { type: Object, required: true } },
    emits: ['update:setting'],
    template:
      '<button class="view" :data-plugin="pluginId" :data-values="JSON.stringify(values)" @click="$emit(\'update:setting\', \'floor\', 7)" />',
  })
  const settings = [
    { key: 'model', type: 'string' as const, title: 'Model', description: '', default: 'a' },
    { key: 'floor', type: 'number' as const, title: 'Floor', description: '', default: 1 },
    { key: 'last', type: 'boolean' as const, title: 'Last', description: '', default: true },
  ]
  const views = [
    { id: 'floor', title: 'Floor' },
    { id: 'top', title: 'Top' },
    { id: 'gone', title: 'Gone' },
  ]
  const mountWith = (registered: Record<string, Component>) => {
    const plugin = createPluginInfo('views.plugin', 'Views', { settings, settingsViews: views })
    const recordedWrites: SettingEdit[] = []
    const installation = createInstallation({
      getSetting: (path) => at({ 'views.plugin': { floor: 3 } }, path),
      write: (edits) => {
        recordedWrites.push(...edits)
      },
      getSettingsViews: () =>
        new Map(Object.entries(registered).map(([id, component]) => [id, { component }])),
      plugins: ref([plugin]),
    })
    const wrapper = mount(PluginDetail, {
      props: {
        settings: installation.settings,
        manager: installation.plugin,
        connections: installation.connections,
        plugin,
      },
    })
    return { wrapper, recordedWrites }
  }

  it('draws a registered view before the settings', () => {
    const { wrapper } = mountWith({ top: View })
    const html = wrapper.html()
    expect(html.indexOf('class="view"')).toBeGreaterThan(-1)
    expect(html.indexOf('class="view"')).toBeLessThan(html.indexOf('Model'))
  })

  it('draws nothing for a view the plugin did not register', () => {
    const { wrapper } = mountWith({})
    expect(wrapper.findAll('.view')).toHaveLength(0)
  })

  it('passes the plugin id and the value of every declared setting', () => {
    const { wrapper } = mountWith({ floor: View })
    const view = wrapper.get('.view')
    expect(view.attributes('data-plugin')).toBe('views.plugin')
    expect(JSON.parse(view.attributes('data-values') ?? '')).toStrictEqual({
      model: 'a',
      floor: 3,
      last: true,
    })
  })

  it('writes the setting a view updates', async () => {
    const { wrapper, recordedWrites } = mountWith({ floor: View })
    await wrapper.get('.view').trigger('click')
    expect(recordedWrites).toStrictEqual([{ at: ['views.plugin', 'floor'], value: '7' }])
  })

  it('shows the settings block for a plugin with only views', () => {
    const plugin = createPluginInfo('views.plugin', 'Views', { settingsViews: views })
    const installation = createInstallation({
      getSettingsViews: () => new Map([['top', { component: View }]]),
      plugins: ref([plugin]),
    })
    const wrapper = mount(PluginDetail, {
      props: {
        settings: installation.settings,
        manager: installation.plugin,
        connections: installation.connections,
        plugin,
      },
    })
    expect(wrapper.findAll('.view')).toHaveLength(1)
  })
})

describe('the data of a plugin', () => {
  const TRANSCRIBER = createPluginInfo('test.transcriber', 'Transcriber')

  /** A plugin page over a folder of `files` files, which a delete empties or rejects with `problem`. */
  const createDataDetail = async (files: number, problem = '') => {
    let count = files
    const getPluginData = vi.fn(async () => count)
    const deletePluginData = vi.fn(async () => {
      if (problem === '') count = 0
      return problem
    })
    const installation = createInstallation({ getPluginData, deletePluginData })
    const wrapper = mount(PluginDetail, {
      props: {
        settings: installation.settings,
        manager: installation.plugin,
        connections: installation.connections,
        plugin: TRANSCRIBER,
      },
    })
    await flushPromises()
    await wrapper
      .findAll('[role="tab"]')
      .find((one) => one.text() === 'Data')
      ?.trigger('click')
    return { wrapper, getPluginData, deletePluginData }
  }

  const findButton = (wrapper: ReturnType<typeof mount>, text: string) =>
    wrapper.findAll('button').find((one) => one.text() === text)

  it('offers no delete where the folder holds no files', async () => {
    const { wrapper, getPluginData } = await createDataDetail(0)

    expect(getPluginData).toHaveBeenCalledWith('test.transcriber')
    expect(findButton(wrapper, words.pluginDataDelete)).toBeUndefined()
  })

  it('deletes the data only after the person confirms', async () => {
    const { wrapper, deletePluginData } = await createDataDetail(3)
    expect(wrapper.get('[role="tabpanel"]').text()).toContain(words.pluginDataDetail(3))

    await findButton(wrapper, words.pluginDataDelete)!.trigger('click')
    expect(deletePluginData).not.toHaveBeenCalled()
    expect(wrapper.get(`[role="alert"][aria-label="${words.pluginDataDelete}"]`).text()).toContain(
      words.pluginDataAsking('Transcriber', 3),
    )

    await findButton(wrapper, words.pluginDataConfirm)!.trigger('click')
    await flushPromises()
    expect(deletePluginData).toHaveBeenCalledExactlyOnceWith('test.transcriber')
    expect(findButton(wrapper, words.pluginDataDelete)).toBeUndefined()
  })

  it('keeps the data when the person keeps it', async () => {
    const { wrapper, deletePluginData } = await createDataDetail(2)

    await findButton(wrapper, words.pluginDataDelete)!.trigger('click')
    await findButton(wrapper, words.pluginDataKeep)!.trigger('click')

    expect(deletePluginData).not.toHaveBeenCalled()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('shows why the data was not deleted', async () => {
    const { wrapper } = await createDataDetail(2, 'the vault is read-only')

    await findButton(wrapper, words.pluginDataDelete)!.trigger('click')
    await findButton(wrapper, words.pluginDataConfirm)!.trigger('click')
    await flushPromises()

    expect(wrapper.findAll('[role="alert"]').map((one) => one.text())).toContain(
      'the vault is read-only',
    )
    expect(findButton(wrapper, words.pluginDataDelete)).toBeDefined()
  })
})

describe('an enum whose value is not offered on this platform', () => {
  const buttons = createPluginInfo('test.plugin', 'Title bar', {
    settings: [
      {
        key: 'buttons',
        type: 'enum',
        title: 'Window buttons',
        description: '',
        default: 'windows',
        options: [
          { value: 'mac', title: 'Left', detail: '' },
          { value: 'windows', title: 'Right', detail: '', when: "app.platform != 'darwin'" },
          { value: 'none', title: 'None', detail: '', when: "app.platform != 'darwin'" },
        ],
      },
    ],
  })

  /** What the closed select shows and what the open list checks, with nothing chosen. */
  const look = async (platform: string, file: JsonObject = {}) => {
    const host = document.createElement('div')
    document.body.append(host)
    const { wrapper, recordedWrites } = createDetail(buttons, file, platform, host)
    const select = wrapper.findComponent(Select)
    const selectedText = wrapper.find('[data-testid="setting-buttons"]').text()
    await select.find('button').trigger('click')
    await flushPromises()
    const checked = [
      ...document.body.querySelectorAll('[role="menuitemradio"][aria-checked="true"]'),
    ].map((one) => one.textContent?.trim())
    wrapper.unmount()
    host.remove()
    return { selectedText, checked, recordedWrites, value: select.props('modelValue') }
  }

  it('shows Left, closed and open, on macOS where the default is Right', async () => {
    const result = await look('darwin')

    expect(result.selectedText).toBe('Left')
    expect(result.checked).toStrictEqual(['Left'])
    expect(result.recordedWrites).toStrictEqual([])
  })

  it('shows Left on macOS where the stored value is None', async () => {
    const result = await look('darwin', { 'test.plugin': { buttons: 'none' } })

    expect(result.selectedText).toBe('Left')
    expect(result.checked).toStrictEqual(['Left'])
    expect(result.recordedWrites).toStrictEqual([])
  })

  it.each(['linux', 'windows'])('shows Right on %s', async (platform) => {
    const result = await look(platform)

    expect(result.selectedText).toBe('Right')
    expect(result.checked).toStrictEqual(['Right'])
  })

  it('shows a stored value that is offered as stored', async () => {
    const result = await look('linux', { 'test.plugin': { buttons: 'none' } })

    expect(result.selectedText).toBe('None')
    expect(result.checked).toStrictEqual(['None'])
  })
})

describe('a setting changed from its default', () => {
  const TUNED = createPluginInfo('test.tuned', 'Tuned', {
    settings: [
      {
        key: 'temperature',
        type: 'number',
        title: 'Temperature',
        description: 'Lower is steadier, higher is more varied',
        default: 0.7,
      },
      { key: 'maxSteps', type: 'number', title: 'Max Steps', description: '', default: 25 },
    ],
  })

  const findResets = (wrapper: ReturnType<typeof createDetail>['wrapper']) =>
    wrapper.findAll('button').filter((one) => one.attributes('aria-label')?.startsWith('Reset'))

  it('offers Reset on its row alone, and Reset writes the default', async () => {
    const { wrapper, recordedWrites } = createDetail(TUNED, {
      'test.tuned': { temperature: 0.4, maxSteps: 25 },
    })

    const resets = findResets(wrapper)
    expect(resets.map((one) => one.attributes('aria-label'))).toStrictEqual([
      'Reset Temperature to its default',
    ])
    await resets[0]!.trigger('click')

    expect(recordedWrites).toStrictEqual([{ at: ['test.tuned', 'temperature'], value: '0.7' }])
  })

  it('offers no Reset while every setting holds its default', () => {
    const { wrapper } = createDetail(TUNED)

    expect(findResets(wrapper)).toHaveLength(0)
  })

  it('describes the control by the whole description the row cuts to one line', () => {
    const { wrapper } = createDetail(TUNED)
    const row = wrapper.findAllComponents(SettingRow)[0]!

    const describedBy = row.getComponent(NumberField).attributes('aria-describedby')
    expect(row.get(`[id="${describedBy}"]`).text()).toBe('Lower is steadier, higher is more varied')
  })

  it('describes no control whose setting has no description', () => {
    const { wrapper } = createDetail(TUNED)
    const row = wrapper.findAllComponents(SettingRow)[1]!

    expect(row.getComponent(NumberField).attributes('aria-describedby')).toBeUndefined()
  })
})

describe('the choice to discover the connections of a plugin', () => {
  const chat = createPluginInfo('org.example.chat', 'Harbor Chat', {
    settings: [
      {
        key: 'agents',
        type: 'connections',
        title: 'Agents',
        description: '',
        default: undefined,
        operations: ['acp.task'],
        max: 5,
        discover: true,
        isShown: true,
      },
    ],
  })

  const findRows = (wrapper: ReturnType<typeof createDetail>['wrapper']) =>
    wrapper.findComponent(ConnectionRows)

  it('discovers while the person has made no choice', () => {
    const { wrapper } = createDetail(chat)

    expect(findRows(wrapper).props('isDiscovering')).toBe(true)
  })

  it('reads the choice the person stored and writes the new one', () => {
    const { wrapper, recordedWrites } = createDetail(chat, {
      'org.example.chat': { discover: { agents: false } },
    })

    expect(findRows(wrapper).props('isDiscovering')).toBe(false)

    findRows(wrapper).vm.$emit('discover', true)

    expect(recordedWrites.map((one) => [one.at, JSON.parse(one.value)])).toStrictEqual([
      [['org.example.chat', 'discover'], { agents: true }],
    ])
  })
})
