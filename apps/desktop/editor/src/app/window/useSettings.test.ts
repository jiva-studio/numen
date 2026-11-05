/**
 * The settings of the window, put together: the lists the palette offers for
 * each setting, and the settings tab reaching each setting through the same
 * piece of code the palette does.
 */
import { describe, expect, it, vi } from 'vitest'
import { createPluginSettingCommands, useSettings } from './useSettings'
import { getNamedIcon } from '@/shared/icons'
import { WORDS } from '@/shared/words'
import type { Model } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'

/** The settings over a vault containing `{}`, and every writer they requested from the log. */
const createSettings = (overrides: Partial<Parameters<typeof useSettings>[0]> = {}) => {
  const writers: string[] = []
  const writerFns: Record<string, ReturnType<typeof vi.fn>> = {}
  const core = {
    getSettings: vi.fn(async () => ({
      written: '{}',
      path: '/vault/.numen/settings.json',
      models: [],
    })),
    writePluginSecret: vi.fn(async () => undefined),
    deletePluginSecret: vi.fn(async () => undefined),
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

/** One model the settings offer, at the setting that names the indexing model. */
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

describe('the settings as they are put together', () => {
  it('gives each setting a message writer under a name of its own', () => {
    const { writers } = createSettings()

    expect(writers).toStrictEqual(['appearance', 'named', 'configured', 'keys', 'plugins'])
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

  it('refreshes the settings in the background after starting and stops when closed', async () => {
    vi.useFakeTimers()
    try {
      const { settings } = createSettings()
      vi.spyOn(settings.appearance, 'start').mockResolvedValue()
      vi.spyOn(settings.sync, 'start').mockResolvedValue()
      const startRest = vi.spyOn(settings.rest, 'start').mockResolvedValue()

      await settings.start()
      expect(settings.appearance.start).toHaveBeenCalledTimes(1)
      expect(settings.sync.start).toHaveBeenCalledTimes(1)
      expect(startRest).toHaveBeenCalledTimes(1)

      vi.advanceTimersByTime(1000)
      expect(startRest).toHaveBeenCalledTimes(1)

      // Starting again at t=1000ms clears the pending 1500ms timer so it does not fire at t=1500ms
      await settings.start()
      expect(startRest).toHaveBeenCalledTimes(2)
      vi.advanceTimersByTime(600)
      expect(startRest).toHaveBeenCalledTimes(2)
      vi.advanceTimersByTime(900)
      expect(startRest).toHaveBeenCalledTimes(3)

      settings.close()
      vi.advanceTimersByTime(20000)
      expect(startRest).toHaveBeenCalledTimes(3)
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
  it('offers the three performance profiles, each with what it does', () => {
    const { settings } = createSettings()

    expect(settings.paletteLists.getStepGroups('app:performance.chooseProfile', '')).toStrictEqual([
      {
        id: 'performance',
        title: WORDS.performance,
        items: [
          { id: 'eco', title: WORDS.eco, detail: WORDS.ecoDetail },
          { id: 'balanced', title: WORDS.balanced, detail: WORDS.balancedDetail },
          { id: 'maximum', title: WORDS.maximum, detail: WORDS.maximumDetail },
        ],
      },
    ])
  })

  it('offers nothing for a command that is over no setting', () => {
    const { settings } = createSettings()
    vi.spyOn(settings.rest, 'getSetting').mockReturnValue('people/own-model')
    vi.spyOn(settings.rest, 'getModelsAt').mockReturnValue([model()])

    expect(settings.paletteLists.getStepGroups('app:note.createChild', '')).toStrictEqual([])
  })

  it('offers the syncing switch the list its own setting holds', () => {
    const { settings } = createSettings()

    expect(settings.paletteLists.getStepGroups('app:note.chooseTitleSync', '')).toStrictEqual(
      settings.sync.getSyncingGroups(),
    )
    expect(
      settings.paletteLists.getStepGroups('app:note.chooseTitleSync', '')[0]?.items,
    ).toHaveLength(2)
  })

  it('previews what the keyboard relies on in an appearance list, and nowhere else', () => {
    const { settings } = createSettings()
    const preview = vi.spyOn(settings.appearance, 'previewItem').mockImplementation(vi.fn())

    settings.paletteLists.previewItem('app:note.chooseTitleSync', 'off')
    settings.paletteLists.previewItem('app:appearance.chooseTheme', 'preset:dracula')

    expect(preview.mock.calls).toStrictEqual([['preset:dracula']])
  })
})

describe('the settings tab', () => {
  it('shows the appearance the window applies', () => {
    const { settings } = createSettings()
    const { installation } = settings.configured.state

    expect(installation.themes).toBe(settings.appearance.list)
    expect(installation.applied).toBe(settings.appearance.applied)
    expect(installation.mode).toBe(settings.appearance.mode)
    expect(installation.isPinned).toBe(settings.appearance.isPinned)
    expect(installation.sizes).toBe(settings.appearance.sized)
    expect(installation.bounds).toBe(settings.appearance.bounds)
  })

  it('chooses an appearance through the palette’s own choice', () => {
    const { settings } = createSettings()
    const choose = vi.spyOn(settings.appearance, 'chooseItem').mockResolvedValue()

    settings.configured.state.installation.choose('preset:dracula')

    expect(choose).toHaveBeenCalledWith('preset:dracula')
  })

  it('reads and writes the syncing switch as the palette rows it is', () => {
    const { settings } = createSettings()
    const { installation } = settings.configured.state
    const sync = vi.spyOn(settings.sync, 'choose').mockResolvedValue()

    expect(installation.isSyncing.value).toBe(settings.sync.isEnabled.value)

    installation.isSyncing.value = true
    installation.isSyncing.value = false

    expect(sync.mock.calls).toStrictEqual([['on'], ['off']])
  })

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
      expect(startRest).toHaveBeenCalledTimes(1)
      settings.close()
    } finally {
      vi.useRealTimers()
    }
  })

  it('generates palette commands for installed plugins with palette: true regardless of isActive', () => {
    createSettings()
    const plugin = {
      id: 'plugin.ocr',
      name: 'OCR Plugin',
      version: '1.0.0',
      description: 'OCR',
      isActive: false,
      capabilities: [],
      secrets: [],
      storedSecrets: [],
      hasFrontend: false,
      hasBackend: false,
      settings: [
        {
          key: 'autoScan',
          type: 'boolean' as const,
          title: 'Auto Scan',
          description: 'Scan on drop',
          default: false,
          palette: true,
          icon: 'scan-text',
        },
        {
          key: 'hiddenSetting',
          type: 'boolean' as const,
          title: 'Hidden',
          description: 'Not in palette',
          default: false,
        },
      ],
    }

    const commands = createPluginSettingCommands([plugin])

    expect(commands).toStrictEqual([
      {
        id: 'plugin-setting:plugin.ocr:autoScan',
        text: 'OCR Plugin: Auto Scan',
        group: 'settings',
        needs: 'choosing',
        icon: getNamedIcon('scan-text'),
        isAvailable: expect.any(Function),
      },
    ])
    expect(commands[0]?.isAvailable({} as never, {} as never)).toBe(true)
  })

  it('returns step groups for plugin boolean and enum settings', () => {
    const { settings } = createSettings()
    settings.rest.plugins.value = [
      {
        id: 'plugin.other',
        name: 'Other Plugin',
        version: '1.0.0',
        description: 'Other',
        isActive: true,
        capabilities: [],
        secrets: [],
        storedSecrets: [],
        hasFrontend: false,
        hasBackend: false,
        settings: [],
      },
      {
        id: 'plugin.ocr',
        name: 'OCR Plugin',
        version: '1.0.0',
        description: 'OCR',
        isActive: true,
        capabilities: [],
        secrets: [],
        storedSecrets: [],
        hasFrontend: false,
        hasBackend: false,
        settings: [
          {
            key: 'autoScan',
            type: 'boolean',
            title: 'Auto Scan',
            description: 'Scan on drop',
            default: true,
            palette: true,
          },
          {
            key: 'engine',
            type: 'enum',
            title: 'Engine',
            description: 'OCR Engine',
            default: 'tesseract',
            options: [
              { value: 'tesseract', title: '', detail: '' },
              { value: 'apple', title: '', detail: '' },
            ],
            palette: true,
          },
          {
            key: 'customText',
            type: 'string',
            title: 'Custom Text',
            description: '',
            default: '',
            palette: true,
          },
        ],
      },
    ]

    expect(settings.paletteLists.getStepGroups('plugin-setting::autoScan', '')).toStrictEqual([])
    expect(settings.paletteLists.getStepGroups('plugin-setting:plugin.ocr:', '')).toStrictEqual([])
    expect(settings.paletteLists.getStepGroups('other:plugin.ocr:autoScan', '')).toStrictEqual([])
    expect(
      settings.paletteLists.getStepGroups('plugin-setting:unknown:autoScan', ''),
    ).toStrictEqual([])
    expect(
      settings.paletteLists.getStepGroups('plugin-setting:plugin.ocr:unknown', ''),
    ).toStrictEqual([])
    expect(
      settings.paletteLists.getStepGroups('plugin-setting:plugin.ocr:customText', ''),
    ).toStrictEqual([])

    const boolGroups = settings.paletteLists.getStepGroups('plugin-setting:plugin.ocr:autoScan', '')
    expect(boolGroups).toStrictEqual([
      {
        id: 'plugin-plugin.ocr-autoScan',
        title: 'Auto Scan',
        items: [
          { id: 'on', title: 'On', detail: WORDS.current, isCurrent: true },
          { id: 'off', title: 'Off' },
        ],
      },
    ])

    const getSettingSpy = vi.spyOn(settings.rest, 'getSetting').mockImplementation((path) => {
      if (path.join('.') === 'plugin.ocr.autoScan') return false
      return undefined
    })
    const boolOffGroups = settings.paletteLists.getStepGroups(
      'plugin-setting:plugin.ocr:autoScan',
      '',
    )
    expect(boolOffGroups).toStrictEqual([
      {
        id: 'plugin-plugin.ocr-autoScan',
        title: 'Auto Scan',
        items: [
          { id: 'on', title: 'On' },
          { id: 'off', title: 'Off', detail: WORDS.current, isCurrent: true },
        ],
      },
    ])
    getSettingSpy.mockRestore()

    const enumGroups = settings.paletteLists.getStepGroups('plugin-setting:plugin.ocr:engine', '')
    expect(enumGroups).toStrictEqual([
      {
        id: 'plugin-plugin.ocr-engine',
        title: 'Engine',
        items: [
          { id: 'tesseract', title: 'tesseract', detail: WORDS.current, isCurrent: true },
          { id: 'apple', title: 'apple' },
        ],
      },
    ])

    expect(
      settings.paletteLists.getStepGroups('app:appearance.chooseTheme', '').length,
    ).toBeGreaterThan(0)
    expect(
      settings.paletteLists.getStepGroups('app:appearance.chooseMode', '').length,
    ).toBeGreaterThan(0)
    expect(
      settings.paletteLists.getStepGroups('app:appearance.chooseInterfaceSize', '').length,
    ).toBeGreaterThan(0)
    expect(
      settings.paletteLists.getStepGroups('app:appearance.chooseTextSize', '').length,
    ).toBeGreaterThan(0)
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
    await (installation.writeSecret(
      'plugin.ocr',
      'apiKey',
      'secret-val',
    ) as unknown as Promise<void>)
    expect(core.writePluginSecret).toHaveBeenCalledWith('plugin.ocr', 'apiKey', 'secret-val')
    expect(startSpy).toHaveBeenCalledTimes(1)

    await (installation.deleteSecret('plugin.ocr', 'apiKey') as unknown as Promise<void>)
    expect(core.deletePluginSecret).toHaveBeenCalledWith('plugin.ocr', 'apiKey')
    expect(startSpy).toHaveBeenCalledTimes(2)

    settings.rest.plugins.value = [
      {
        id: 'plugin.other',
        name: 'Other Plugin',
        version: '1.0.0',
        description: '',
        isActive: true,
        capabilities: [],
        secrets: [],
        storedSecrets: [],
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
        storedSecrets: [],
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
      await (installation.writeSecret(
        'plugin.ocr',
        'apiKey2',
        'secret-val-2',
      ) as unknown as Promise<void>)
      expect(startSpy).toHaveBeenCalledTimes(1)
      vi.advanceTimersByTime(1500)
      expect(startSpy).toHaveBeenCalledTimes(2)

      startSpy.mockClear()
      vi.mocked(core.choosePluginPath).mockResolvedValueOnce({
        path: '/folder',
        isChosen: true,
        problem: '',
      })
      expect(await installation.choosePath('plugin.ocr', 'dir')).toBe('')
      expect(startSpy).toHaveBeenCalledTimes(1)
      vi.advanceTimersByTime(1500)
      expect(startSpy).toHaveBeenCalledTimes(2)

      vi.mocked(core.choosePluginPath).mockResolvedValueOnce({
        path: '',
        isChosen: false,
        problem: 'cancelled',
      })
      expect(await installation.choosePath('plugin.ocr', 'dir')).toBe('cancelled')
      expect(startSpy).toHaveBeenCalledTimes(2)
      settings.close()
    } finally {
      vi.useRealTimers()
    }

    vi.mocked(core.choosePluginPath).mockRejectedValueOnce(new Error('dialog crashed'))
    expect(await installation.choosePath('plugin.ocr', 'dir')).toBe(
      'Numen is not responding; retrying',
    )

    expect(await installation.getPluginData?.('plugin.ocr')).toBe(5)
    expect(core.getPluginData).toHaveBeenCalledWith('plugin.ocr')

    expect(await installation.deletePluginData?.('plugin.ocr')).toBe('')
    expect(core.deletePluginData).toHaveBeenCalledWith('plugin.ocr')

    vi.mocked(core.deletePluginData).mockRejectedValueOnce(new Error('permission denied'))
    expect(await installation.deletePluginData?.('plugin.ocr')).toBe(
      'Numen is not responding; retrying',
    )

    vi.spyOn(settings.sync, 'choose').mockRejectedValueOnce(new Error('sync write failed'))
    installation.isSyncing.value = true
    await vi.waitFor(() =>
      expect(writerFns.configured).toHaveBeenCalledWith(
        'Numen is not responding; retrying',
        'error',
      ),
    )
    settings.close()
  })
})
