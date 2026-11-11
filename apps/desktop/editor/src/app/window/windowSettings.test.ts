/**
 * The settings of the window, put together: the lists the palette offers for
 * each setting, and the settings tab reaching each setting through the same
 * piece of code the palette does.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createWindowSettingsPart } from '@/testing/windowSettings'
import { WORDS } from '@/shared/ui/words'
import {
  createSettingsStore,
  SETTINGS_WORDS,
  type Model,
  type PluginInfo,
  type PluginSettingInfo,
} from '@/entities/settings'
import { installSettingsCore, type SettingsCore } from '@/testing/settingsWire'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'
import { createLostConnection, type CoreAnswers } from '@numen/core-client/testing'

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

const createSettings = (
  overrides: Partial<Parameters<typeof createWindowSettingsPart>[0]> = {},
  others: CoreAnswers = {},
) => {
  const writers: string[] = []
  const writerFns: Record<string, ReturnType<typeof vi.fn<MessageWriter>>> = {}
  const core = {
    getSettings: vi.fn(async () => ({
      written: '{}',
      path: '/vault/.numen/settings.json',
      models: [],
    })),
    updateSettings: vi.fn(async () => undefined),
    watchSettings: vi.fn(async function* () {
      yield* []
    }),
    createSecret: vi.fn(async () => 's0a1b2c3d'),
    renameSecret: vi.fn(async () => undefined),
    writeSecret: vi.fn(async () => undefined),
    moveSecret: vi.fn(async () => undefined),
    renameSecretGroup: vi.fn(async () => undefined),
    allowPluginSecret: vi.fn(async () => undefined),
    denyPluginSecret: vi.fn(async () => undefined),
    deleteSecret: vi.fn(async () => undefined),
    deleteSecretGroup: vi.fn(async () => undefined),
    listSecretStates: vi.fn(async () => new Map<string, boolean>()),
    retryPlugin: vi.fn(async () => ''),
    choosePluginPath: vi.fn(async () => ({ path: '/picked', isChosen: true, problem: '' })),
    getPluginData: vi.fn(async () => 5),
    deletePluginData: vi.fn(async () => undefined),
  } as unknown as SettingsCore
  restore?.()
  restore = installSettingsCore(core, others)
  const log = {
    getWriter: (name: string) => {
      writers.push(name)
      const fn = vi.fn()
      writerFns[name] = fn
      return fn
    },
  } as unknown as MessageLog
  const handle = { openTab: vi.fn(async () => ''), openTabAdjacent: vi.fn(async () => '') }
  const windowTabs = { groups: { handle } } as unknown as WindowTabs
  const onSizeChanged = vi.fn()
  const settings = createWindowSettingsPart({
    rest: createSettingsStore(SETTINGS_WORDS, (text, level) => writerFns.configured?.(text, level)),
    words: WORDS,
    log,
    windowTabs,
    onSizeChanged,
    getSettingsViews: () => new Map(),
    getPlatform: () => 'linux',
    ...overrides,
  })
  return { settings, writers, writerFns, handle, core, onSizeChanged }
}

const model = (overrides: Partial<Model> = {}): Model => ({
  namedAt: ['indexing', 'embedding', 'model', 'name'],
  name: 'e5-small-int8',
  title: 'E5 small',
  group: 'On this machine',
  isDefault: false,
  presence: 'present',
  writes: [],
  reasoningLevels: [],
  defaultReasoning: '',
  ...overrides,
})

const SHELL_DECLARATION: readonly PluginSettingInfo[] = [
  {
    key: 'appearance.theme',
    type: 'enum',
    title: 'Theme',
    description: '',
    default: 'preset:numen',
    palette: true,
    icon: 'palette',
    preview: true,
    activeOptions: [{ value: 'preset:numen', title: 'numen', detail: '' }],
    value: 'preset:numen',
  },
  {
    key: 'appearance.mode',
    type: 'enum',
    title: 'Light or dark',
    description: '',
    default: 'system',
    palette: true,
    preview: true,
    options: [{ value: 'system', title: 'System', detail: '' }],
  },
  {
    key: 'appearance.interfaceScale',
    type: 'range',
    title: 'Interface size',
    description: '',
    default: 1,
    palette: true,
    preview: true,
    min: 0.8,
    max: 2,
    step: 0.1,
  },
  {
    key: 'appearance.textScale',
    type: 'range',
    title: 'Reading size',
    description: '',
    default: 1,
    palette: true,
    preview: true,
    min: 0.8,
    max: 1.75,
    step: 0.1,
  },
  {
    key: 'naming.syncTitleAndFilename',
    type: 'boolean',
    title: 'Sync title and filename',
    description: '',
    default: true,
    palette: true,
  },
  {
    key: 'performance.profile',
    type: 'enum',
    title: 'Performance profile',
    description: '',
    default: 'balanced',
    palette: true,
    options: [
      { value: 'eco', title: 'Eco', detail: '' },
      { value: 'balanced', title: 'Balanced', detail: '' },
    ],
  },
]

const SCANNER: PluginInfo = {
  id: 'md.example.scanner',
  name: 'Scanner',
  version: '1.0.0',
  description: '',
  isActive: true,
  capabilities: [],
  secrets: [],
  hasFrontend: false,
  hasBackend: false,
  settings: [
    {
      key: 'autoScan',
      type: 'boolean',
      title: 'Auto scan',
      description: '',
      default: false,
      palette: true,
    },
  ],
}

describe('the settings as they are put together', () => {
  it('gives each setting a message writer under a name of its own', () => {
    const { writers } = createSettings()

    expect([...writers].sort()).toStrictEqual(['appearance', 'configured', 'keys', 'plugins'])
  })

  it('opens the settings tab as kind of tab', () => {
    const { settings } = createSettings()

    expect(settings.kinds).toStrictEqual([settings.configured.kind])
  })

  it('puts the appearance away when it is put away', async () => {
    vi.useFakeTimers()
    try {
      const { settings, onSizeChanged } = createSettings()

      settings.appearance.preview.keyboard.previewSize('interfaceScale', 1.2)
      settings.close()
      vi.advanceTimersByTime(1000)
      await Promise.resolve()

      expect(onSizeChanged).not.toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })

  it('reads the settings once when it starts and never again on a timer', async () => {
    vi.useFakeTimers()
    try {
      const listThemes = vi.fn(() => ({
        themes: [],
        applied: 'preset:numen',
        mode: 0,
        interfaceScale: 1,
        textScale: 1,
        interfaceScaleBounds: { min: 0.8, max: 2 },
        textScaleBounds: { min: 0.8, max: 1.75 },
      }))
      const { settings, core } = createSettings(
        {},
        {
          theme: {
            listThemes,
            readTheme: () => ({ css: '' }),
            async *watchThemes() {
              yield* []
            },
          },
        },
      )

      await settings.start()
      expect(listThemes).toHaveBeenCalledTimes(1)
      expect(core.getSettings).toHaveBeenCalledTimes(1)

      vi.advanceTimersByTime(20000)
      expect(core.getSettings).toHaveBeenCalledTimes(1)

      settings.close()
    } finally {
      vi.useRealTimers()
    }
  })

  it('notifies onSizeChanged when appearance.sized changes', async () => {
    const { settings, onSizeChanged } = createSettings()
    settings.appearance.state.settings.value = { interfaceScale: 110, textScale: 105 }
    await Promise.resolve()
    expect(onSizeChanged).toHaveBeenCalledTimes(1)
  })
})

describe('the lists the palette offers for a setting', () => {
  it('offers nothing for a command that is over no setting', () => {
    const { settings } = createSettings()
    settings.rest.state.target.value = {
      indexing: { embedding: { model: { name: 'people/own-model' } } },
    }
    settings.rest.state.models.value = [model()]

    expect(settings.paletteLists.getStepGroups('app:note.createChild', '')).toStrictEqual([])
  })

  it('offers the six settings of the shell from the declaration the core sends, and no command made by hand', () => {
    const { settings } = createSettings()
    settings.rest.state.app.value = SHELL_DECLARATION

    const ids = settings.settingPalette.getCommands().map((one) => one.id)

    expect(ids).toStrictEqual([
      'setting:app:appearance.theme',
      'setting:app:appearance.mode',
      'setting:app:appearance.interfaceScale',
      'setting:app:appearance.textScale',
      'setting:app:naming.syncTitleAndFilename',
      'setting:app:performance.profile',
    ])
    for (const old of [
      'app:appearance.chooseTheme',
      'app:appearance.chooseMode',
      'app:appearance.chooseInterfaceSize',
      'app:appearance.chooseTextSize',
      'app:note.chooseTitleSync',
      'app:performance.chooseProfile',
    ]) {
      expect(settings.paletteLists.getStepGroups(old, ''), old).toStrictEqual([])
    }
  })

  it('offers the settings of the installed plugins together with those of the shell', () => {
    const { settings } = createSettings()
    settings.rest.state.app.value = SHELL_DECLARATION
    settings.rest.state.plugins.value = [SCANNER]

    const ids = settings.settingPalette.getCommands().map((one) => one.id)

    expect(ids).toContain('setting:app:performance.profile')
    expect(ids).toContain('setting:md.example.scanner:autoScan')
  })

  it('draws the rows of every command from the adapter, whichever owner declares the setting', () => {
    const { settings } = createSettings()
    settings.rest.state.app.value = SHELL_DECLARATION
    settings.rest.state.plugins.value = [SCANNER]

    for (const command of [
      'setting:app:performance.profile',
      'setting:app:naming.syncTitleAndFilename',
      'setting:md.example.scanner:autoScan',
    ]) {
      const groups = settings.paletteLists.getStepGroups(command, '')
      expect(groups.length, command).toBe(1)
      expect(groups).toStrictEqual(settings.settingPalette.lists.getStepGroups(command, ''))
    }
  })

  it('writes the row chosen under the path of the setting through the settings the window holds', async () => {
    const { settings, core } = createSettings()
    settings.rest.state.app.value = SHELL_DECLARATION

    await settings.settingPalette.choose('setting:app:performance.profile', 'eco')

    expect(core.updateSettings).toHaveBeenCalledWith([
      { at: ['app', 'performance', 'profile'], value: '"eco"' },
    ])
  })
})

describe('the settings tab', () => {
  it('reads and writes the rest of the file through the settings the window holds', async () => {
    vi.useFakeTimers()
    try {
      const { settings, core } = createSettings()
      const { installation } = settings.configured.state
      vi.mocked(core.getSettings).mockResolvedValue({
        written: JSON.stringify({ app: { performance: { profile: 'eco' } } }),
        path: '/vault/.numen/settings.json',
        models: [model()],
      })
      const edits = [{ at: ['app', 'performance', 'profile'], value: '"eco"' }]

      await (installation.settings.write(edits) as unknown as Promise<void>)
      vi.advanceTimersByTime(1500)

      expect(installation.settings.getSetting(['app', 'performance', 'profile'])).toBe('eco')
      expect(
        installation.settings.getModels(['indexing', 'embedding', 'model', 'name']),
      ).toStrictEqual([model()])
      expect(core.updateSettings).toHaveBeenCalledWith(edits)
      expect(core.getSettings).toHaveBeenCalledTimes(1)
      settings.close()
    } finally {
      vi.useRealTimers()
    }
  })

  it('delegates settings tab installation actions, path choosing, plugin data, and error reporting', async () => {
    const getSettingsViews = vi.fn(() => new Map())
    const { settings, core, writerFns } = createSettings({ getSettingsViews })
    const { installation } = settings.configured.state

    expect(installation.plugin.getSettingsViews).toBe(getSettingsViews)
    expect(installation.plugin.catalog).toBeDefined()

    settings.rest.state.target.value = { a: { b: 'committed-val' } }
    expect(installation.settings.getConfirmedSetting?.(['a', 'b'])).toBe('committed-val')
    expect(installation.settings.isPending?.(['a', 'b'])).toBe(false)

    await (installation.secrets.edit.writeSecret(
      's0a1b2c3d',
      'secret-val',
    ) as unknown as Promise<void>)
    expect(core.writeSecret).toHaveBeenCalledWith('s0a1b2c3d', 'secret-val')
    expect(core.getSettings).toHaveBeenCalledTimes(1)

    await (installation.secrets.edit.deleteSecret('s0a1b2c3d') as unknown as Promise<void>)
    expect(core.deleteSecret).toHaveBeenCalledWith('s0a1b2c3d')
    expect(core.getSettings).toHaveBeenCalledTimes(2)

    await (installation.secrets.access.allowPluginSecret(
      's0a1b2c3d',
      'plugin.ocr',
      'apiKey',
      'openrouter',
    ) as unknown as Promise<void>)
    expect(core.allowPluginSecret).toHaveBeenCalledWith(
      's0a1b2c3d',
      'plugin.ocr',
      'apiKey',
      'openrouter',
    )
    await (installation.secrets.access.denyPluginSecret(
      's0a1b2c3d',
      'plugin.ocr',
      'apiKey',
      'openrouter',
    ) as unknown as Promise<void>)
    expect(core.denyPluginSecret).toHaveBeenCalledWith(
      's0a1b2c3d',
      'plugin.ocr',
      'apiKey',
      'openrouter',
    )
    expect(core.getSettings).toHaveBeenCalledTimes(4)

    const ocrPlugins: readonly PluginInfo[] = [
      {
        id: 'plugin.other',
        name: 'Other Plugin',
        version: '1.0.0',
        description: '',
        isActive: true,
        capabilities: [],
        secrets: [],
        hasFrontend: false,
        hasBackend: false,
        settings: [],
      },
      {
        id: 'plugin.ocr',
        name: 'OCR Plugin',
        version: '1.0.0',
        description: '',
        isActive: true,
        capabilities: [],
        secrets: [],
        hasFrontend: false,
        hasBackend: false,
        settings: [],
      },
    ]
    vi.mocked(core.getSettings).mockImplementation(async () => ({
      written: '{}',
      path: '/vault/.numen/settings.json',
      models: [],
      plugins: ocrPlugins,
    }))
    settings.rest.state.plugins.value = ocrPlugins
    vi.mocked(core.retryPlugin).mockResolvedValueOnce('Failed to start')
    await (installation.plugin.retryPlugin('plugin.ocr') as unknown as Promise<void>)
    expect(core.retryPlugin).toHaveBeenCalledWith('plugin.ocr')
    expect(writerFns.plugins).toHaveBeenCalledWith('OCR Plugin stopped: failed to start', 'warning')

    vi.mocked(core.retryPlugin).mockResolvedValueOnce('Failed unknown')
    await (installation.plugin.retryPlugin('unknown.id') as unknown as Promise<void>)
    expect(writerFns.plugins).toHaveBeenCalledWith('unknown.id stopped: failed unknown', 'warning')

    vi.useFakeTimers()
    try {
      vi.mocked(core.getSettings).mockClear()
      await (installation.secrets.edit.writeSecret(
        's0a1b2c3d',
        'secret-val-2',
      ) as unknown as Promise<void>)
      expect(core.getSettings).toHaveBeenCalledTimes(1)
      vi.advanceTimersByTime(1500)
      expect(core.getSettings).toHaveBeenCalledTimes(1)

      vi.mocked(core.getSettings).mockClear()
      vi.mocked(core.choosePluginPath).mockResolvedValueOnce({
        path: '/folder',
        isChosen: true,
        problem: '',
      })
      expect(await installation.plugin.choosePath('plugin.ocr', 'dir')).toBe('')
      expect(core.getSettings).toHaveBeenCalledTimes(1)
      vi.advanceTimersByTime(1500)
      expect(core.getSettings).toHaveBeenCalledTimes(1)

      vi.mocked(core.choosePluginPath).mockResolvedValueOnce({
        path: '',
        isChosen: false,
        problem: 'cancelled',
      })
      expect(await installation.plugin.choosePath('plugin.ocr', 'dir')).toBe('cancelled')
      expect(core.getSettings).toHaveBeenCalledTimes(1)
      settings.close()
    } finally {
      vi.useRealTimers()
    }

    vi.mocked(core.choosePluginPath).mockRejectedValueOnce(createLostConnection('dialog crashed'))
    expect(await installation.plugin.choosePath('plugin.ocr', 'dir')).toBe(
      'Numen is not responding; retrying',
    )

    expect(await installation.plugin.files.getPluginData('plugin.ocr')).toBe(5)
    expect(core.getPluginData).toHaveBeenCalledWith('plugin.ocr')

    expect(await installation.plugin.files.deletePluginData('plugin.ocr')).toBe('')
    expect(core.deletePluginData).toHaveBeenCalledWith('plugin.ocr')

    vi.mocked(core.deletePluginData).mockRejectedValueOnce(
      createLostConnection('permission denied'),
    )
    expect(await installation.plugin.files.deletePluginData('plugin.ocr')).toBe(
      'Numen is not responding; retrying',
    )

    vi.mocked(core.updateSettings).mockRejectedValueOnce(createLostConnection('write failed'))
    const failed = installation.settings.write([
      { at: ['app', 'performance', 'profile'], value: '"eco"' },
    ]) as unknown as Promise<void>
    await failed.catch(() => undefined)
    await vi.waitFor(() =>
      expect(writerFns.configured).toHaveBeenCalledWith(
        'Setting not saved: Numen is not responding; retrying',
        'error',
      ),
    )
    settings.close()
  })
})

describe('the plugin catalog', () => {
  it('writes the reason a plugin source failed to the window log', async () => {
    const { settings, writerFns } = createSettings(
      {},
      {
        pluginCatalog: {
          listCatalog: () => ({
            entries: [],
            sourceErrors: [
              {
                source: 'local',
                message: 'duplicate plugin id "sample.notes" in /plugins/a and /plugins/b',
              },
            ],
          }),
        },
      },
    )

    settings.rest.state.plugins.value = [SCANNER]

    await vi.waitFor(() =>
      expect(writerFns.configured).toHaveBeenCalledWith(
        'The plugin source local failed: duplicate plugin id "sample.notes" in /plugins/a and /plugins/b',
        'error',
      ),
    )
    settings.close()
  })

  it('writes the reason the catalog is unavailable to the window log', async () => {
    const { settings, writerFns } = createSettings(
      {},
      {
        pluginCatalog: {
          listCatalog: () => {
            throw createLostConnection('the catalog is down')
          },
        },
      },
    )

    settings.rest.state.plugins.value = [SCANNER]

    await vi.waitFor(() =>
      expect(writerFns.configured).toHaveBeenCalledWith(
        'The plugin catalog is unavailable: Numen is not responding; retrying',
        'error',
      ),
    )
    settings.close()
  })
})
