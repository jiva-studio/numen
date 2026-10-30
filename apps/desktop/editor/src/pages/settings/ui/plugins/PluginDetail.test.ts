import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import PluginDetail from './PluginDetail.vue'
import SettingRow from '../setting-row/SettingRow.vue'
import { SecretField } from '../secret-field'
import { createInstallation, createPluginInfo } from '@/testing/installation'
import type { Installation, SettingsTabState } from '../../types'
import { getSettingAt as at, type PluginInfo, type SettingEdit } from '@/entities/settings'
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

const createDetail = (plugin: PluginInfo, file: Record<string, unknown> = {}) => {
  const recordedWrites: SettingEdit[] = []
  const retryPlugin = vi.fn()
  const writeSecret = vi.fn()
  const deleteSecret = vi.fn()
  const installation: Installation = createInstallation({
    getSetting: (path) => at(file, path),
    write: (edits) => {
      recordedWrites.push(...edits)
    },
    writeSecret,
    deleteSecret,
    retryPlugin,
    plugins: ref([plugin]),
  })
  const state: SettingsTabState = { installation }
  const wrapper = mount(PluginDetail, {
    props: { state, plugin },
  })
  return {
    wrapper,
    recordedWrites,
    retryPlugin,
    writeSecret,
    deleteSecret,
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

    expect(wrapper.find('.plugin-detail__title').text()).toBe('Super Engine v1.2.3')
    expect(wrapper.find('.plugin-detail__description').text()).toBe('Engine description text')
  })

  it('renders back button and clicking it emits back', async () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine')
    const { wrapper } = createDetail(plugin)

    const topBackButton = wrapper.find('.plugin-detail__back')
    expect(topBackButton.exists()).toBe(true)
    expect(topBackButton.text()).toContain(words.backToPlugins)

    await topBackButton.trigger('click')
    expect(wrapper.emitted('back')).toHaveLength(1)
  })

  it('toggles master enabled switch and writes setting', async () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine')
    const { wrapper, recordedWrites } = createDetail(plugin)

    const masterSwitch = wrapper.find('.plugin-detail__master-switch').findComponent(Switch)
    expect(masterSwitch.exists()).toBe(true)
    expect(masterSwitch.props('modelValue')).toBe(true)

    masterSwitch.vm.$emit('update:modelValue', false)

    expect(recordedWrites).toStrictEqual([
      { at: ['plugins', 'test.plugin', 'enabled'], value: 'false' },
    ])
  })

  it('reads master enabled switch state when disabled in settings', () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine')
    const { wrapper } = createDetail(plugin, {
      plugins: { 'test.plugin': { enabled: false } },
    })

    const masterSwitch = wrapper.find('.plugin-detail__master-switch').findComponent(Switch)
    expect(masterSwitch.props('modelValue')).toBe(false)
  })

  it('renders failure card, calls retryPlugin on retry, and writes disabled on disable', async () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine', {
      failure: {
        reason: 'Timed out while starting',
        detail: 'handshake timeout after 10s',
        failedAt: 1_700_000_000_000,
      },
    })
    const { wrapper, retryPlugin, recordedWrites } = createDetail(plugin)

    const stoppedCard = wrapper.find('.plugin-detail__stopped')
    expect(stoppedCard.exists()).toBe(true)
    expect(stoppedCard.text()).toContain('Timed out while starting')
    expect(stoppedCard.find('details summary').text()).toBe('Details')
    expect(stoppedCard.find('details code').text()).toBe('handshake timeout after 10s')

    const buttons = stoppedCard.findAll('button')
    const retryButton = buttons.find((btn) => btn.text() === words.pluginRetry)
    const disableButton = buttons.find((btn) => btn.text() === words.pluginDisable)

    expect(retryButton?.exists()).toBe(true)
    expect(disableButton?.exists()).toBe(true)

    await retryButton!.trigger('click')
    expect(retryPlugin).toHaveBeenCalledExactlyOnceWith('test.plugin')

    await disableButton!.trigger('click')
    expect(recordedWrites).toStrictEqual([
      { at: ['plugins', 'test.plugin', 'enabled'], value: 'false' },
    ])
  })

  it('renders AI consent switch and writes aiConsent change', () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine', {
      consent: 'allowed',
    })
    const { wrapper, recordedWrites } = createDetail(plugin)

    const rows = wrapper.findAllComponents(SettingRow)
    const consentRow = rows.find((r) => r.props('at') === 'ai-consent-test.plugin')
    expect(consentRow).toBeDefined()
    expect(consentRow!.text()).toContain(words.consentAllowed)

    const consentSwitch = consentRow!.findComponent(Switch)
    expect(consentSwitch.props('modelValue')).toBe(true)

    consentSwitch.vm.$emit('update:modelValue', false)

    expect(recordedWrites).toStrictEqual([
      { at: ['plugins', 'test.plugin', 'aiConsent'], value: 'false' },
    ])
  })

  it('saves and deletes secret fields', () => {
    const plugin = createPluginInfo('test.plugin', 'Super Engine', {
      secrets: [{ name: 'AUTH_TOKEN', title: 'AUTH_TOKEN', description: '' }],
      storedSecrets: ['AUTH_TOKEN'],
    })
    const { wrapper, writeSecret, deleteSecret } = createDetail(plugin)

    const secretField = wrapper.findComponent(SecretField)
    expect(secretField.exists()).toBe(true)
    expect(secretField.props('name')).toBe('AUTH_TOKEN')
    expect(secretField.props('isStored')).toBe(true)

    secretField.vm.$emit('save', 'secret-key-123')
    expect(writeSecret).toHaveBeenCalledExactlyOnceWith(
      'test.plugin',
      'AUTH_TOKEN',
      'secret-key-123',
    )

    secretField.vm.$emit('delete')
    expect(deleteSecret).toHaveBeenCalledExactlyOnceWith('test.plugin', 'AUTH_TOKEN')
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
      at: ['plugins', 'test.plugin', 'autoStart'],
      value: 'true',
    })

    // NumberField
    const numberRow = rows.find((r) => r.props('at') === 'test.plugin-retryCount')
    expect(numberRow).toBeDefined()
    const numberComp = numberRow!.findComponent(NumberField)
    expect(numberComp.props('modelValue')).toBe(3)
    numberComp.vm.$emit('commit', 5)
    expect(recordedWrites).toContainEqual({
      at: ['plugins', 'test.plugin', 'retryCount'],
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
      at: ['plugins', 'test.plugin', 'confidence'],
      value: '90',
    })

    // Select
    const enumRow = rows.find((r) => r.props('at') === 'test.plugin-logLevel')
    expect(enumRow).toBeDefined()
    const selectComp = enumRow!.findComponent(Select)
    expect(selectComp.props('modelValue')).toBe('info')
    selectComp.vm.$emit('update:modelValue', 'debug')
    expect(recordedWrites).toContainEqual({
      at: ['plugins', 'test.plugin', 'logLevel'],
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
      at: ['plugins', 'test.plugin', 'endpoint'],
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
      { at: ['plugins', 'test.plugin', 'endpoint'], value: '"http://127.0.0.1:9/v1"' },
    ])
  })

  it('renders clean detail view with switch when plugin has no additional settings', () => {
    const plugin = createPluginInfo('test.empty', 'Empty Plugin')
    const { wrapper } = createDetail(plugin)

    expect(wrapper.find('.plugin-detail__title').text()).toContain('Empty Plugin')
    expect(wrapper.find('.plugin-detail__master-switch').exists()).toBe(true)
    expect(wrapper.find('.plugin-detail__empty').exists()).toBe(false)
  })

  it('puts the back button first, above the title', () => {
    const { wrapper } = createDetail(createPluginInfo('test.plugin', 'Super Engine'))
    const root = wrapper.find('.plugin-detail').element
    expect(root.firstElementChild?.classList.contains('plugin-detail__back')).toBe(true)
  })

  it('renders a header card without a state badge and expands README details on Read more', async () => {
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

    expect(wrapper.find('.plugin-detail__state').exists()).toBe(false)
    expect(wrapper.find('.plugin-detail__description').text()).toBe('Short summary line.')
    expect(wrapper.find('.plugin-detail__more').exists()).toBe(false)

    const toggle = wrapper.find('.plugin-detail__toggle')
    expect(toggle.exists()).toBe(true)
    expect(toggle.text()).toBe(words.readMore)

    await toggle.trigger('click')
    expect(wrapper.find('.plugin-detail__more').text()).toContain('Detailed documentation body.')
    expect(toggle.text()).toBe(words.showLess)
  })

  it('draws no stopped details disclosure when failure detail is empty', () => {
    const failure = { reason: 'Failed to start', detail: '', failedAt: 0 }
    const { wrapper } = createDetail(createPluginInfo('a', 'A', { failure }))
    expect(wrapper.find('.plugin-detail__stopped details').exists()).toBe(false)
  })
})

describe('PluginDetail with settings that follow one another', () => {
  const option = (value: string, title = '', detail = '') => ({ value, title, detail })
  const LINKED = createPluginInfo('test.linked', 'Linked', {
    secrets: [
      {
        name: 'apiKey',
        title: 'API key',
        description: '',
        when: { setting: 'provider', is: ['remote'] },
      },
    ],
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
  const findRow = (wrapper: ReturnType<typeof mount>, key: string) =>
    wrapper.findAllComponents(SettingRow).find((r) => r.props('at') === `test.linked-${key}`)

  it('lists the languages of the current model with their detail subtitles and no narrative origin text', () => {
    const { wrapper } = createDetail(LINKED)
    const row = findRow(wrapper, 'language')!
    expect(row.findComponent(Select).props('choices')).toEqual([
      { id: 'auto', text: 'Detected by the model', detail: 'Among 3 languages' },
      { id: 'en', text: 'English' },
    ])
    expect(row.props('detail')).toBe('Spoken language')
    expect(row.find('.settings__origin').exists()).toBe(false)
  })

  it('shows the one language of a model as fixed', () => {
    const small = createDetail(LINKED, { plugins: { 'test.linked': { model: 'small' } } })
    const row = findRow(small.wrapper, 'language')!
    expect(row.findComponent(Select).exists()).toBe(false)
    expect(row.find('.settings__fixed').text()).toBe('English')
  })

  it('falls back to the active provider option when a string setting holds an option of another provider', () => {
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
    const { wrapper } = createDetail(embeddingsPlugin, {
      plugins: {
        'md.numen.engines.embeddings': {
          provider: 'local',
          model: 'openai/text-embedding-3-small',
        },
      },
    })
    const row = wrapper
      .findAllComponents(SettingRow)
      .find((r) => r.props('at') === 'md.numen.engines.embeddings-model')!
    expect(row.findComponent(Select).props('modelValue')).toBe('Xenova/bge-m3')
  })

  it('shows the settings and the key of a provider only while it is chosen, placing the key after provider', () => {
    const local = createDetail(LINKED)
    expect(findRow(local.wrapper, 'url')).toBeUndefined()
    expect(local.wrapper.findComponent(SecretField).exists()).toBe(false)

    const remote = createDetail(LINKED, { plugins: { 'test.linked': { provider: 'remote' } } })
    expect(findRow(remote.wrapper, 'url')).toBeDefined()
    expect(remote.wrapper.findComponent(SecretField).props('name')).toBe('API key')
    const rowIds = remote.wrapper.findAllComponents(SettingRow).map((r) => r.props('at'))
    const providerIndex = rowIds.indexOf('test.linked-provider')
    const keyIndex = rowIds.indexOf('key-test.linked-apiKey')
    const urlIndex = rowIds.indexOf('test.linked-url')
    expect(providerIndex).toBeGreaterThanOrEqual(0)
    expect(keyIndex).toBe(providerIndex + 1)
    expect(urlIndex).toBeGreaterThan(keyIndex)
  })

  it('chooses a path with the picker, says why it was not taken, and clears it', async () => {
    const { wrapper, installation, recordedWrites } = createDetail(LINKED, {
      plugins: { 'test.linked': { dictionary: '/home/a/dicts/ru/letters.txt' } },
    })
    const choosePath = vi.fn(async () => 'letters.onnx: wrong type of file, expected .txt')
    installation.choosePath = choosePath
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
      at: ['plugins', 'test.linked', 'dictionary'],
      value: '""',
    })
  })

  it('hides the path label when no path is chosen and supports multiple paths', async () => {
    const emptyDetail = createDetail(LINKED)
    const emptyRow = findRow(emptyDetail.wrapper, 'dictionary')!
    expect(emptyRow.find('[data-testid="setting-path"]').exists()).toBe(false)

    const acpPlugin: PluginInfo = {
      ...createPluginInfo('test.acp', 'ACP Agents'),
      settings: [
        {
          key: 'executable_path',
          type: 'file',
          title: 'Executable Paths',
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
      at: ['plugins', 'test.acp', 'executable_path'],
      value: '"/usr/local/bin/opencode"',
    })
    const addButton = acpRow.findAll('button').find((btn) => btn.text() === words.pathAdd)
    expect(addButton).toBeDefined()
  })
})

describe('the data of a plugin', () => {
  const TRANSCRIBER = createPluginInfo('test.transcriber', 'Transcriber')

  /** A plugin page over a folder of `files` files, which a delete empties or refuses with `problem`. */
  const createDataDetail = async (files: number, problem = '') => {
    let count = files
    const getPluginData = vi.fn(async () => count)
    const deletePluginData = vi.fn(async () => {
      if (problem === '') count = 0
      return problem
    })
    const installation = createInstallation({ getPluginData, deletePluginData })
    const wrapper = mount(PluginDetail, { props: { state: { installation }, plugin: TRANSCRIBER } })
    await flushPromises()
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
    expect(wrapper.find('.plugin-detail__data').exists()).toBe(true)
    expect(wrapper.text()).toContain(words.pluginDataDetail(3))

    await findButton(wrapper, words.pluginDataDelete)!.trigger('click')
    expect(deletePluginData).not.toHaveBeenCalled()
    expect(wrapper.find('.plugin-detail__asking').text()).toContain(
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
    expect(wrapper.find('.plugin-detail__asking').exists()).toBe(false)
  })

  it('shows why the data was not deleted', async () => {
    const { wrapper } = await createDataDetail(2, 'the vault is read-only')

    await findButton(wrapper, words.pluginDataDelete)!.trigger('click')
    await findButton(wrapper, words.pluginDataConfirm)!.trigger('click')
    await flushPromises()

    expect(wrapper.find('.plugin-detail__problem').text()).toBe('the vault is read-only')
    expect(findButton(wrapper, words.pluginDataDelete)).toBeDefined()
  })
})
