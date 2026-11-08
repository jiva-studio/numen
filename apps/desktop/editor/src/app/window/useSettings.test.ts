/**
 * The settings of the window, put together: the lists the palette offers for
 * each setting, and the settings tab reaching each setting through the same
 * piece of code the palette does.
 */
import { describe, expect, it, vi } from 'vitest'
import { useSettings } from './useSettings'
import { WORDS } from '@/shared/words'
import type { Model, PluginInfo, PluginSettingInfo } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'
import {
  ListCatalogResponseSchema,
  asFailure,
  asValue,
  create,
  type PluginCatalog,
} from '@numen/core-client'
import { createLostConnection } from '@numen/core-client/testing'

const createSettings = (overrides: Partial<Parameters<typeof useSettings>[0]> = {}) => {
  const writers: string[] = []
  const writerFns: Record<string, ReturnType<typeof vi.fn>> = {}
  const core = {
    getSettings: vi.fn(async () => ({
      written: '{}',
      path: '/vault/.numen/settings.json',
      models: [],
    })),
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
  } as unknown as SettingsPort
  const log = {
    getWriter: (name: string) => {
      writers.push(name)
      const fn = vi.fn()
      writerFns[name] = fn
      return fn
    },
  } as unknown as MessageLog
  const handle = { openTab: vi.fn(async () => ''), openTabAdjacent: vi.fn(async () => '') }
  const windowTabs = { handle } as unknown as ReturnType<typeof useWindowTabs>
  const onSizeChanged = vi.fn()
  const settings = useSettings({
    core,
    words: WORDS,
    log,
    windowTabs,
    onSizeChanged,
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

    expect(writers).toStrictEqual(['appearance', 'configured', 'keys', 'plugins'])
  })

  it('opens the settings tab as kind of tab', () => {
    const { settings } = createSettings()

    expect(settings.kinds).toStrictEqual([settings.configured.kind])
  })

  it('puts the appearance away when it is put away', () => {
    const { settings } = createSettings()
    const close = vi.spyOn(settings.appearance, 'close')

    settings.close()

    expect(close).toHaveBeenCalledOnce()
  })

  it('reads the settings once when it starts and never again on a timer', async () => {
    vi.useFakeTimers()
    try {
      const { settings } = createSettings()
      vi.spyOn(settings.appearance, 'start').mockResolvedValue()
      const startRest = vi.spyOn(settings.rest, 'start').mockResolvedValue()

      await settings.start()
      expect(settings.appearance.start).toHaveBeenCalledTimes(1)
      expect(startRest).toHaveBeenCalledTimes(1)

      vi.advanceTimersByTime(20000)
      expect(startRest).toHaveBeenCalledTimes(1)

      settings.close()
    } finally {
      vi.useRealTimers()
    }
  })

  it('notifies onSizeChanged when appearance.sized changes', async () => {
    const { settings, onSizeChanged } = createSettings()
    settings.appearance.settings.value = { interfaceScale: 110, textScale: 105 }
    await Promise.resolve()
    expect(onSizeChanged).toHaveBeenCalledTimes(1)
  })
})

describe('the lists the palette offers for a setting', () => {
  it('offers nothing for a command that is over no setting', () => {
    const { settings } = createSettings()
    vi.spyOn(settings.rest, 'getSetting').mockReturnValue('people/own-model')
    vi.spyOn(settings.rest, 'getModelsAt').mockReturnValue([model()])

    expect(settings.paletteLists.getStepGroups('app:note.createChild', '')).toStrictEqual([])
  })

  it('offers the six settings of the shell from the declaration the core sends, and no command made by hand', () => {
    const { settings } = createSettings()
    settings.rest.app.value = SHELL_DECLARATION

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
    settings.rest.app.value = SHELL_DECLARATION
    settings.rest.plugins.value = [SCANNER]

    const ids = settings.settingPalette.getCommands().map((one) => one.id)

    expect(ids).toContain('setting:app:performance.profile')
    expect(ids).toContain('setting:md.example.scanner:autoScan')
  })

  it('draws the rows of every command from the adapter, whichever owner declares the setting', () => {
    const { settings } = createSettings()
    settings.rest.app.value = SHELL_DECLARATION
    settings.rest.plugins.value = [SCANNER]

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
    const { settings } = createSettings()
    settings.rest.app.value = SHELL_DECLARATION
    const write = vi.spyOn(settings.rest, 'writeSettings').mockResolvedValue()

    await settings.settingPalette.choose('setting:app:performance.profile', 'eco')

    expect(write).toHaveBeenCalledWith([{ at: ['app', 'performance', 'profile'], value: '"eco"' }])
  })
})

describe('the settings tab', () => {
  it('reads and writes the rest of the file through the settings the window holds', async () => {
    vi.useFakeTimers()
    try {
      const { settings } = createSettings()
      const { installation } = settings.configured.state
      vi.spyOn(settings.rest, 'getSetting').mockReturnValue('eco')
      vi.spyOn(settings.rest, 'getModelsAt').mockReturnValue([model()])
      const startRest = vi.spyOn(settings.rest, 'start').mockResolvedValue()
      const write = vi.spyOn(settings.rest, 'writeSettings').mockResolvedValue()
      const edits = [{ at: ['app', 'performance', 'profile'], value: '"eco"' }]

      await (installation.write(edits) as unknown as Promise<void>)
      vi.advanceTimersByTime(1500)

      expect(installation.getSetting(['app', 'performance', 'profile'])).toBe('eco')
      expect(installation.getModels(['indexing'])).toStrictEqual([model()])
      expect(write).toHaveBeenCalledWith(edits)
      expect(startRest).toHaveBeenCalledTimes(0)
      settings.close()
    } finally {
      vi.useRealTimers()
    }
  })

  it('delegates settings tab installation actions, path choosing, plugin data, and error reporting', async () => {
    const getSettingsViews = vi.fn(() => new Map())
    const catalog = {
      listPlugins: vi.fn(async () => ({ plugins: [] })),
    } as unknown as NonNullable<Parameters<typeof useSettings>[0]['catalog']>
    const { settings, core, writerFns } = createSettings({ getSettingsViews, catalog })
    const { installation } = settings.configured.state

    expect(installation.getSettingsViews).toBe(getSettingsViews)
    expect(installation.catalog).toBeDefined()

    vi.spyOn(settings.rest, 'getCommittedSetting').mockReturnValue('committed-val')
    vi.spyOn(settings.rest, 'isPending').mockReturnValue(true)
    expect(installation.getConfirmedSetting?.(['a', 'b'])).toBe('committed-val')
    expect(settings.rest.getCommittedSetting).toHaveBeenCalledWith(['a', 'b'])
    expect(installation.isPending?.(['a', 'b'])).toBe(true)
    expect(settings.rest.isPending).toHaveBeenCalledWith(['a', 'b'])

    const startSpy = vi.spyOn(settings.rest, 'start').mockResolvedValue()
    await (installation.writeSecret('s0a1b2c3d', 'secret-val') as unknown as Promise<void>)
    expect(core.writeSecret).toHaveBeenCalledWith('s0a1b2c3d', 'secret-val')
    expect(startSpy).toHaveBeenCalledTimes(1)

    await (installation.deleteSecret('s0a1b2c3d') as unknown as Promise<void>)
    expect(core.deleteSecret).toHaveBeenCalledWith('s0a1b2c3d')
    expect(startSpy).toHaveBeenCalledTimes(2)

    await (installation.allowPluginSecret(
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
    await (installation.denyPluginSecret(
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
    expect(startSpy).toHaveBeenCalledTimes(4)

    settings.rest.plugins.value = [
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
    vi.mocked(core.retryPlugin).mockResolvedValueOnce('Failed to start')
    await (installation.retryPlugin('plugin.ocr') as unknown as Promise<void>)
    expect(core.retryPlugin).toHaveBeenCalledWith('plugin.ocr')
    expect(writerFns.plugins).toHaveBeenCalledWith('OCR Plugin stopped: failed to start', 'warning')

    vi.mocked(core.retryPlugin).mockResolvedValueOnce('Failed unknown')
    await (installation.retryPlugin('unknown.id') as unknown as Promise<void>)
    expect(writerFns.plugins).toHaveBeenCalledWith('unknown.id stopped: failed unknown', 'warning')

    vi.useFakeTimers()
    try {
      startSpy.mockClear()
      await (installation.writeSecret('s0a1b2c3d', 'secret-val-2') as unknown as Promise<void>)
      expect(startSpy).toHaveBeenCalledTimes(1)
      vi.advanceTimersByTime(1500)
      expect(startSpy).toHaveBeenCalledTimes(1)

      startSpy.mockClear()
      vi.mocked(core.choosePluginPath).mockResolvedValueOnce({
        path: '/folder',
        isChosen: true,
        problem: '',
      })
      expect(await installation.choosePath('plugin.ocr', 'dir')).toBe('')
      expect(startSpy).toHaveBeenCalledTimes(1)
      vi.advanceTimersByTime(1500)
      expect(startSpy).toHaveBeenCalledTimes(1)

      vi.mocked(core.choosePluginPath).mockResolvedValueOnce({
        path: '',
        isChosen: false,
        problem: 'cancelled',
      })
      expect(await installation.choosePath('plugin.ocr', 'dir')).toBe('cancelled')
      expect(startSpy).toHaveBeenCalledTimes(1)
      settings.close()
    } finally {
      vi.useRealTimers()
    }

    vi.mocked(core.choosePluginPath).mockRejectedValueOnce(createLostConnection('dialog crashed'))
    expect(await installation.choosePath('plugin.ocr', 'dir')).toBe(
      'Numen is not responding; retrying',
    )

    expect(await installation.getPluginData?.('plugin.ocr')).toBe(5)
    expect(core.getPluginData).toHaveBeenCalledWith('plugin.ocr')

    expect(await installation.deletePluginData?.('plugin.ocr')).toBe('')
    expect(core.deletePluginData).toHaveBeenCalledWith('plugin.ocr')

    vi.mocked(core.deletePluginData).mockRejectedValueOnce(
      createLostConnection('permission denied'),
    )
    expect(await installation.deletePluginData?.('plugin.ocr')).toBe(
      'Numen is not responding; retrying',
    )

    vi.spyOn(settings.rest, 'writeSettings').mockRejectedValueOnce(
      createLostConnection('write failed'),
    )
    const failed = installation.write([
      { at: ['app', 'performance', 'profile'], value: '"eco"' },
    ]) as unknown as Promise<void>
    await failed.catch(() => undefined)
    await vi.waitFor(() =>
      expect(writerFns.configured).toHaveBeenCalledWith(
        'Numen is not responding; retrying',
        'error',
      ),
    )
    settings.close()
  })
})

describe('the plugin catalog', () => {
  const createCatalog = (listing: Awaited<ReturnType<PluginCatalog['list']>>): PluginCatalog => ({
    list: vi.fn(async () => listing),
    get: vi.fn(),
    findByTabKind: vi.fn(),
  })

  it('writes the reason a plugin source failed to the window log', async () => {
    const catalog = createCatalog(
      asValue(
        create(ListCatalogResponseSchema, {
          sourceErrors: [
            {
              source: 'local',
              message: 'duplicate plugin id "sample.notes" in /plugins/a and /plugins/b',
            },
          ],
        }),
      ),
    )
    const { settings, writerFns } = createSettings({ catalog })

    settings.rest.plugins.value = [SCANNER]

    await vi.waitFor(() =>
      expect(writerFns.configured).toHaveBeenCalledWith(
        'The plugin source local failed: duplicate plugin id "sample.notes" in /plugins/a and /plugins/b',
        'error',
      ),
    )
    settings.close()
  })

  it('writes the reason the catalog is unavailable to the window log', async () => {
    const catalog = createCatalog(
      asFailure({ kind: 'unavailable', message: 'Numen is busy; try again' }),
    )
    const { settings, writerFns } = createSettings({ catalog })

    settings.rest.plugins.value = [SCANNER]

    await vi.waitFor(() =>
      expect(writerFns.configured).toHaveBeenCalledWith(
        'The plugin catalog is unavailable: Numen is busy; try again',
        'error',
      ),
    )
    settings.close()
  })
})
