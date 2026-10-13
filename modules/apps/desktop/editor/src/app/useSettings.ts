/**
 * Settings configuration, theme dressing, and palette lists for the window.
 */
import { computed, watch } from 'vue'
import { themes } from '@/entities/settings'
import {
  APPEARANCE,
  APPEARANCE_COMMANDS,
  NESTING,
  INTERFACE_SCALE,
  MODE,
  OFF,
  ON,
  HEADINGS,
  SYNCING,
  syncSetting,
  TEXT_SCALE,
  useNestingSetting,
  createWindowAppearance,
} from '@/features/settings-commands'
import { createReviewSetting } from '@/entities/settings'
import { createSettingsStore } from '@/entities/settings'
import { useSettingsTab, paths, createChoices } from '@/pages/settings'
import type { SelectChoice } from '@numen/ui'
import { createTextEditorTabKind } from '@/pages/text-editor'
import type { PaletteLists } from '@/features/command-palette'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import { WORDS } from '@/shared/words'
import type { useWindowTabs } from '@/entities/tab'

type Words = typeof WORDS

export interface SettingsDeps {
  core: SettingsPort
  words: Words
  log: MessageLog
  windowTabs: ReturnType<typeof useWindowTabs>
  onSizeChanged: () => void
}

function getPerformanceStepGroups(words: Words) {
  return [
    {
      id: 'performance',
      title: words.performance,
      items: [
        { id: 'eco', title: words.eco, detail: words.ecoDetail },
        { id: 'balanced', title: words.balanced, detail: words.balancedDetail },
        { id: 'maximum', title: words.maximum, detail: words.maximumDetail },
        { id: 'custom', title: words.custom, detail: words.customDetail },
      ],
    },
  ]
}

function getIndexingModelStepGroups(rest: ReturnType<typeof createSettingsStore>, words: Words) {
  const models = rest.getModelsAt(paths.indexingModel)
  const current = rest.getSetting(paths.indexingModel)
  const choices = createChoices(models, typeof current === 'string' ? current : '', words)

  const groups: Record<string, SelectChoice[]> = {}
  for (const choice of choices) {
    const groupName = choice.group ?? 'Other'
    const list = groups[groupName] ?? []
    list.push(choice)
    groups[groupName] = list
  }

  return Object.entries(groups).map(([groupName, groupChoices]) => ({
    id: `indexingModel-${groupName}`,
    title: groupName === 'Other' ? words.indexingModel : groupName,
    items: groupChoices.map((c) => ({
      id: c.id,
      title: c.text,
      ...(c.detail ? { detail: c.detail } : {}),
    })),
  }))
}

interface SettingsTabConfigDeps {
  dressed: ReturnType<typeof createWindowAppearance>
  sync: ReturnType<typeof syncSetting>
  nesting: ReturnType<typeof useNestingSetting>
  dayBegins: ReturnType<typeof createReviewSetting>
  rest: ReturnType<typeof createSettingsStore>
  file: ReturnType<typeof createTextEditorTabKind>
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps) {
  const { dressed, sync, nesting, dayBegins, rest, file } = deps
  return {
    themes: dressed.list,
    applied: dressed.applied,
    mode: dressed.mode,
    isPinned: dressed.isPinned,
    sizes: dressed.sized,
    bounds: dressed.bounds,
    choose: (item: string) => void dressed.chooseItem(item),
    isSyncing: computed({
      get: () => sync.isEnabled.value,
      set: (on: boolean) => void sync.choose(on ? ON : OFF),
    }),
    isNesting: computed({
      get: () => nesting.isNesting.value,
      set: (on: boolean) => void nesting.choose(on ? ON : OFF),
    }),
    headings: nesting.headings,
    headingsBounds: nesting.ends,
    chooseHeadings: (count: number) => void nesting.chooseCount(`${count}`),
    dayStartTime: dayBegins.starts,
    latestDayStartTime: dayBegins.latest,
    chooseDayStartTime: (hour: number) => void dayBegins.choose(hour),
    getSetting: (at: readonly string[]) => rest.getSetting(at),
    getModels: (at: readonly string[]) => rest.getModelsAt(at),
    write: (written: readonly SettingEdit[]) => void rest.writeSettings(written),
    file: rest.path,
    openFile: () => file.openSettingsFile(),
  }
}

interface PaletteListsDeps {
  dressed: ReturnType<typeof createWindowAppearance>
  sync: ReturnType<typeof syncSetting>
  nesting: ReturnType<typeof useNestingSetting>
  rest: ReturnType<typeof createSettingsStore>
  words: SettingsWords
}

function createSettingsPaletteLists(deps: PaletteListsDeps): PaletteLists {
  const { dressed, sync, nesting, rest, words } = deps
  return {
    getStepGroups: (command, typed) => {
      if (command === APPEARANCE) return dressed.getThemeGroups()
      if (command === MODE) return dressed.getModeGroups()
      if (command === INTERFACE_SCALE || command === TEXT_SCALE)
        return dressed.getSizeGroups(command, typed)
      if (command === SYNCING) return sync.getSyncingGroups()
      if (command === NESTING) return nesting.getNestingGroups()
      if (command === HEADINGS) return nesting.getHeadingsGroups()
      if (command === 'performanceProfile') return getPerformanceStepGroups(words)
      if (command === 'indexingModel') return getIndexingModelStepGroups(rest, words)
      return []
    },
    previewItem: (command, item) => {
      if (APPEARANCE_COMMANDS.includes(command)) dressed.previewItem(item)
    },
  }
}

export function useSettings({ core, words, log, windowTabs, onSizeChanged }: SettingsDeps) {
  const dayBegins = createReviewSetting(core, words, log.getWriter('reviewed'))
  const nesting = useNestingSetting(core, words, log.getWriter('nesting'))
  const dressed = createWindowAppearance({ core: themes, words, write: log.getWriter('worn') })
  const sync = syncSetting(core, words, log.getWriter('named'))
  const rest = createSettingsStore(core, words, log.getWriter('configured'))

  const file = createTextEditorTabKind(windowTabs.handle, core, () => void rest.start())

  const configured = useSettingsTab(
    windowTabs.handle,
    createSettingsTabConfig({ dressed, sync, nesting, dayBegins, rest, file }),
  )

  watch(dressed.sized, () => onSizeChanged())

  const kept = createSettingsPaletteLists({ dressed, sync, nesting, rest, words })

  const kinds = [configured.kind, file.kind]

  const start = async () => {
    await dayBegins.start()
    void dressed.start()
    void sync.start()
    void nesting.start()
    void rest.start()
  }

  const close = () => {
    dressed.close()
  }

  return {
    dayBegins,
    nesting,
    dressed,
    sync,
    oneName: sync,
    rest,
    file,
    configured,
    kept,
    kinds,
    start,
    close,
  }
}
