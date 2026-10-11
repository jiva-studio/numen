/**
 * The settings of the window, put together: the lists the palette offers for
 * each setting, and the settings tab reaching each setting through the same
 * piece of code the palette does.
 */
import { describe, expect, it, vi } from 'vitest'
import { useSettings } from './useSettings'
import { WORDS } from '@/shared/words'
import type { Model } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'

/** The settings over a vault holding `{}`, and every writer they asked the log for. */
const createSettings = () => {
  const writers: string[] = []
  const core = {
    getSettingsFile: async () => ({ written: '{}', path: '/vault/.numen/settings.json' }),
    saveSettingsFile: async () => ({ isChanged: false }),
  } as unknown as SettingsPort
  const log = {
    getWriter: (name: string) => {
      writers.push(name)
      return () => {}
    },
  } as unknown as MessageLog
  const windowTabs = {
    handle: { openTab: async () => '', openTabBeside: async () => '' },
  } as unknown as ReturnType<typeof useWindowTabs>
  const settings = useSettings({ core, words: WORDS, log, windowTabs, onSizeChanged: () => {} })
  return { settings, writers }
}

/** One model the settings offer, at the setting that names the indexing model. */
const model = (over: Partial<Model> = {}): Model => ({
  namedAt: ['indexing', 'embedding', 'model', 'name'],
  name: 'e5-small-int8',
  title: 'E5 small',
  shelf: 'On this machine',
  isDefault: false,
  presence: 'present',
  writes: [],
  ...over,
})

/** A moment for what a call set going to land. */
const settle = () => new Promise((done) => setTimeout(done, 0))

describe('the settings as they are put together', () => {
  it('gives each setting a message writer under a name of its own', () => {
    const { writers } = createSettings()

    expect(writers).toStrictEqual(['reviewed', 'hanging', 'worn', 'named', 'configured'])
  })

  it('opens the settings tab and the settings file as kinds of tab', () => {
    const { settings } = createSettings()

    expect(settings.kinds).toStrictEqual([settings.configured.kind, settings.file.kind])
  })

  it('puts the appearance away when it is put away', () => {
    const { settings } = createSettings()
    const close = vi.spyOn(settings.dressed, 'close')

    settings.close()

    expect(close).toHaveBeenCalledOnce()
  })

  it('reads every setting again once the settings file is saved', async () => {
    const { settings } = createSettings()
    const start = vi.spyOn(settings.rest, 'start').mockResolvedValue()
    const tab = settings.file.kind.open('')
    await settle()

    tab.type('{ "a": 1 }')
    await tab.save()

    expect(start).toHaveBeenCalledOnce()
  })
})

describe('the lists the palette offers for a setting', () => {
  it('offers the four performance profiles, each with what it does', () => {
    const { settings } = createSettings()

    expect(settings.kept.getStepGroups('performanceProfile', '')).toStrictEqual([
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

  it('offers the indexing models by shelf, with a model the file names first', () => {
    const { settings } = createSettings()
    const at = vi.spyOn(settings.rest, 'getSetting').mockReturnValue('people/own-model')
    vi.spyOn(settings.rest, 'getModelsAt').mockReturnValue([
      model(),
      model({ name: 'bge-small', title: 'BGE small', shelf: '', presence: 'nothing to fetch' }),
    ])

    expect(settings.kept.getStepGroups('indexingModel', '')).toStrictEqual([
      {
        id: `indexingModel-${WORDS.owned}`,
        title: WORDS.owned,
        items: [{ id: 'people/own-model', title: 'own-model', detail: 'people/own-model' }],
      },
      {
        id: 'indexingModel-On this machine',
        title: 'On this machine',
        items: [{ id: 'e5-small-int8', title: 'E5 small', detail: WORDS.present }],
      },
      {
        id: 'indexingModel-Other',
        title: WORDS.indexingModel,
        items: [{ id: 'bge-small', title: 'BGE small' }],
      },
    ])
    expect(at).toHaveBeenCalledWith(['indexing', 'embedding', 'model', 'name'])
  })

  it('offers only the models there are where the file names no model in words', () => {
    const { settings } = createSettings()
    vi.spyOn(settings.rest, 'getSetting').mockReturnValue(5)
    vi.spyOn(settings.rest, 'getModelsAt').mockReturnValue([model()])

    expect(settings.kept.getStepGroups('indexingModel', '')).toStrictEqual([
      {
        id: 'indexingModel-On this machine',
        title: 'On this machine',
        items: [{ id: 'e5-small-int8', title: 'E5 small', detail: WORDS.present }],
      },
    ])
  })

  it('offers nothing for a command that is over no setting', () => {
    const { settings } = createSettings()
    vi.spyOn(settings.rest, 'getSetting').mockReturnValue('people/own-model')
    vi.spyOn(settings.rest, 'getModelsAt').mockReturnValue([model()])

    expect(settings.kept.getStepGroups('child', '')).toStrictEqual([])
  })

  it('offers each switch the list its own setting holds', () => {
    const { settings } = createSettings()

    expect(settings.kept.getStepGroups('syncing', '')).toStrictEqual(
      settings.sync.getSyncingGroups(),
    )
    expect(settings.kept.getStepGroups('hanging', '')).toStrictEqual(
      settings.hungParts.getHangingGroups(),
    )
    expect(settings.kept.getStepGroups('parts', '')).toStrictEqual(
      settings.hungParts.getPartsGroups(),
    )
    expect(settings.kept.getStepGroups('syncing', '')[0]?.items).toHaveLength(2)
  })

  it('previews what the keyboard stands on in an appearance list, and nowhere else', () => {
    const { settings } = createSettings()
    const preview = vi.spyOn(settings.dressed, 'previewItem').mockImplementation(() => {})

    settings.kept.previewItem('syncing', 'off')
    settings.kept.previewItem('appearance', 'preset:dracula')

    expect(preview.mock.calls).toStrictEqual([['preset:dracula']])
  })
})

describe('the settings tab', () => {
  it('shows the appearance the window wears', () => {
    const { settings } = createSettings()
    const { installation } = settings.configured.state

    expect(installation.themes).toBe(settings.dressed.list)
    expect(installation.applied).toBe(settings.dressed.applied)
    expect(installation.mode).toBe(settings.dressed.mode)
    expect(installation.isPinned).toBe(settings.dressed.isPinned)
    expect(installation.sizes).toBe(settings.dressed.sized)
    expect(installation.bounds).toBe(settings.dressed.bounds)
  })

  it('chooses an appearance through the palette’s own choice', () => {
    const { settings } = createSettings()
    const choose = vi.spyOn(settings.dressed, 'chooseItem').mockResolvedValue()

    settings.configured.state.installation.choose('preset:dracula')

    expect(choose).toHaveBeenCalledWith('preset:dracula')
  })

  it('reads and writes the two switches as the palette rows they are', () => {
    const { settings } = createSettings()
    const { installation } = settings.configured.state
    const sync = vi.spyOn(settings.sync, 'choose').mockResolvedValue()
    const hang = vi.spyOn(settings.hungParts, 'choose').mockResolvedValue()
    settings.sync.isEnabled.value = false
    settings.hungParts.isHanging.value = false

    expect(installation.isSyncing.value).toBe(false)
    expect(installation.isHanging.value).toBe(false)

    installation.isSyncing.value = true
    installation.isHanging.value = true
    installation.isSyncing.value = false
    installation.isHanging.value = false

    expect(sync.mock.calls).toStrictEqual([['on'], ['off']])
    expect(hang.mock.calls).toStrictEqual([['on'], ['off']])
  })

  it('chooses a count of parts and the hour a day begins through the palette’s own choices', () => {
    const { settings } = createSettings()
    const { installation } = settings.configured.state
    const count = vi.spyOn(settings.hungParts, 'chooseCount').mockResolvedValue()
    const hour = vi.spyOn(settings.dayBegins, 'choose').mockResolvedValue()

    installation.chooseParts(4)
    installation.chooseDayStarts('05:30')

    expect(count).toHaveBeenCalledWith('4')
    expect(hour).toHaveBeenCalledWith('05:30')
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
    const open = vi.spyOn(settings.file, 'openSettingsFile').mockImplementation(() => {})

    settings.configured.state.installation.openFile()

    expect(open).toHaveBeenCalledOnce()
  })
})
