/**
 * The settings of the window, put together: the lists the palette offers for
 * each setting, and the settings tab reaching each setting through the same
 * piece of code the palette does.
 */
import { describe, expect, it, vi } from 'vitest'
import { createPluginSettingCommands, useSettings } from './useSettings'
import { WORDS } from '@/shared/words'
import type { Model } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'

/** The settings over a vault holding `{}`, and every writer they asked the log for. */
const createSettings = () => {
  const writers: string[] = []
  const core = {
    getSettings: async () => ({ written: '{}', path: '/vault/.numen/settings.json', models: [] }),
    getSettingsFile: async () => ({ written: '{}', path: '/vault/.numen/settings.json' }),
    saveSettingsFile: async () => ({ isChanged: false }),
  } as unknown as SettingsPort
  const log = {
    getWriter: (name: string) => {
      writers.push(name)
      return vi.fn()
    },
  } as unknown as MessageLog
  const handle = { openTab: vi.fn(async () => ''), openTabBeside: vi.fn(async () => '') }
  const windowTabs = { handle } as unknown as ReturnType<typeof useWindowTabs>
  const settings = useSettings({ core, words: WORDS, log, windowTabs, onSizeChanged: vi.fn() })
  return { settings, writers, handle }
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

    expect(writers).toStrictEqual(['nesting', 'worn', 'named', 'configured', 'keys', 'plugins'])
  })

  it('opens the settings tab as kind of tab', () => {
    const { settings } = createSettings()

    expect(settings.kinds).toStrictEqual([settings.configured.kind, settings.file.kind])
  })

  it('puts the appearance away when it is put away', () => {
    const { settings } = createSettings()
    const close = vi.spyOn(settings.appearance, 'close')

    settings.close()

    expect(close).toHaveBeenCalledOnce()
  })

  it('opens the settings file alongside the current tab', async () => {
    const { settings, handle } = createSettings()
    await settings.rest.start()
    settings.file.openSettingsFile()

    expect(handle.openTabBeside).toHaveBeenCalledWith('settings-file')
  })
})

describe('the lists the palette offers for a setting', () => {
  it('offers the four performance profiles, each with what it does', () => {
    const { settings } = createSettings()

    expect(settings.paletteLists.getStepGroups('performanceProfile', '')).toStrictEqual([
      {
        id: 'performance',
        title: WORDS.performance,
        items: [
          { id: 'eco', title: WORDS.eco, detail: WORDS.ecoDetail },
          { id: 'balanced', title: WORDS.balanced, detail: WORDS.balancedDetail },
          { id: 'maximum', title: WORDS.maximum, detail: WORDS.maximumDetail },
          { id: 'custom', title: WORDS.custom, detail: WORDS.customDetail },
        ],
      },
    ])
  })

  it('offers nothing for a command that is over no setting', () => {
    const { settings } = createSettings()
    vi.spyOn(settings.rest, 'getSetting').mockReturnValue('people/own-model')
    vi.spyOn(settings.rest, 'getModelsAt').mockReturnValue([model()])

    expect(settings.paletteLists.getStepGroups('child', '')).toStrictEqual([])
  })

  it('offers each switch the list its own setting holds', () => {
    const { settings } = createSettings()

    expect(settings.paletteLists.getStepGroups('syncing', '')).toStrictEqual(
      settings.sync.getSyncingGroups(),
    )
    expect(settings.paletteLists.getStepGroups('nesting', '')).toStrictEqual(
      settings.nesting.getNestingGroups(),
    )
    expect(settings.paletteLists.getStepGroups('headings', '')).toStrictEqual(
      settings.nesting.getHeadingsGroups(),
    )
    expect(settings.paletteLists.getStepGroups('syncing', '')[0]?.items).toHaveLength(2)
  })

  it('previews what the keyboard relies on in an appearance list, and nowhere else', () => {
    const { settings } = createSettings()
    const preview = vi.spyOn(settings.appearance, 'previewItem').mockImplementation(vi.fn())

    settings.paletteLists.previewItem('syncing', 'off')
    settings.paletteLists.previewItem('appearance', 'preset:dracula')

    expect(preview.mock.calls).toStrictEqual([['preset:dracula']])
  })
})

describe('the settings tab', () => {
  it('shows the appearance the window wears', () => {
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

  it('reads and writes the two switches as the palette rows they are', () => {
    const { settings } = createSettings()
    const { installation } = settings.configured.state
    const sync = vi.spyOn(settings.sync, 'choose').mockResolvedValue()
    const nest = vi.spyOn(settings.nesting, 'choose').mockResolvedValue()
    settings.sync.isEnabled.value = false
    settings.nesting.isNesting.value = false

    expect(installation.isSyncing.value).toBe(false)
    expect(installation.isNesting.value).toBe(false)

    installation.isSyncing.value = true
    installation.isNesting.value = true
    installation.isSyncing.value = false
    installation.isNesting.value = false

    expect(sync.mock.calls).toStrictEqual([['on'], ['off']])
    expect(nest.mock.calls).toStrictEqual([['on'], ['off']])
  })

  it('chooses a count of headings through the palette’s own choices', () => {
    const { settings } = createSettings()
    const { installation } = settings.configured.state
    const count = vi.spyOn(settings.nesting, 'chooseCount').mockResolvedValue()

    installation.chooseHeadings(4)

    expect(count).toHaveBeenCalledWith('4')
  })

  it('reads and writes the rest of the file through the settings the window holds', () => {
    const { settings } = createSettings()
    const { installation } = settings.configured.state
    vi.spyOn(settings.rest, 'getSetting').mockReturnValue('eco')
    vi.spyOn(settings.rest, 'getModelsAt').mockReturnValue([model()])
    const write = vi.spyOn(settings.rest, 'writeSettings').mockResolvedValue()
    const edits = [{ at: ['performance', 'profile'], value: '"eco"' }]

    installation.write(edits)

    expect(installation.getSetting(['performance', 'profile'])).toBe('eco')
    expect(installation.getModels(['indexing'])).toStrictEqual([model()])
    expect(write).toHaveBeenCalledWith(edits)
  })

  it('opens the settings file in a tab of its own', () => {
    const { settings } = createSettings()
    const open = vi.spyOn(settings.file, 'openSettingsFile').mockImplementation(vi.fn())

    settings.configured.state.installation.openFile()

    expect(open).toHaveBeenCalledOnce()
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
      consent: 'none' as const,
      hasFrontend: false,
      settings: [
        {
          key: 'autoScan',
          type: 'boolean' as const,
          title: 'Auto Scan',
          description: 'Scan on drop',
          default: false,
          palette: true,
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
        group: 'window',
        needs: 'choosing',
        isOffered: expect.any(Function),
      },
    ])
  })

  it('returns step groups for plugin boolean and enum settings', () => {
    const { settings } = createSettings()
    settings.rest.plugins.value = [
      {
        id: 'plugin.ocr',
        name: 'OCR Plugin',
        version: '1.0.0',
        description: 'OCR',
        isActive: true,
        capabilities: [],
        secrets: [],
        storedSecrets: [],
        consent: 'none',
        hasFrontend: false,
        settings: [
          {
            key: 'autoScan',
            type: 'boolean',
            title: 'Auto Scan',
            description: 'Scan on drop',
            default: false,
            palette: true,
          },
          {
            key: 'engine',
            type: 'enum',
            title: 'Engine',
            description: 'OCR Engine',
            default: 'tesseract',
            options: ['tesseract', 'apple'],
            palette: true,
          },
        ],
      },
    ]

    const boolGroups = settings.paletteLists.getStepGroups('plugin-setting:plugin.ocr:autoScan', '')
    expect(boolGroups).toStrictEqual([
      {
        id: 'plugin-plugin.ocr-autoScan',
        title: 'Auto Scan',
        items: [
          { id: 'on', title: 'On' },
          { id: 'off', title: 'Off', detail: WORDS.current, isCurrent: true },
        ],
      },
    ])

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
  })
})
