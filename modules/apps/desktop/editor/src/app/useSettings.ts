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
import { createReviewSetting, type PluginInfo, type SettingEdit } from '@/entities/settings'
import { createSecretStore, createSettingsStore } from '@/entities/settings'
import { useSettingsTab, paths, createChoices } from '@/pages/settings'
import type { SelectChoice } from '@numen/ui'
import type { Command, PaletteLists } from '@/features/command-palette'
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

interface PluginGroupContext {
  readonly id: string
  readonly title: string
  readonly words: Words
}

function createPluginBooleanGroup(ctx: PluginGroupContext, isOn: boolean) {
  return [
    {
      id: ctx.id,
      title: ctx.title,
      items: [
        {
          id: 'on',
          title: ctx.words.on,
          ...(isOn ? { detail: ctx.words.current, isCurrent: true } : {}),
        },
        {
          id: 'off',
          title: ctx.words.off,
          ...(!isOn ? { detail: ctx.words.current, isCurrent: true } : {}),
        },
      ],
    },
  ]
}

function createPluginEnumGroup(
  ctx: PluginGroupContext,
  current: string,
  options: readonly string[],
) {
  return [
    {
      id: ctx.id,
      title: ctx.title,
      items: options.map((opt) => ({
        id: opt,
        title: opt,
        ...(opt === current ? { detail: ctx.words.current, isCurrent: true } : {}),
      })),
    },
  ]
}

function findPluginSetting(plugins: readonly PluginInfo[], command: string) {
  const [, pluginId, key] = command.split(':')
  if (!pluginId || !key) return null
  const plugin = plugins.find((p) => p.id === pluginId)
  const setting = plugin?.settings.find((s) => s.key === key)
  if (!setting) return null
  return { pluginId, key, setting }
}

function getPluginStepGroups(
  command: string,
  rest: ReturnType<typeof createSettingsStore>,
  words: Words,
) {
  const match = findPluginSetting(rest.plugins.value, command)
  if (!match) return []
  const { pluginId, key, setting } = match
  const current = rest.getSetting(['plugins', pluginId, key])
  const ctx: PluginGroupContext = { id: `plugin-${pluginId}-${key}`, title: setting.title, words }

  if (setting.type === 'boolean') {
    const isOn = current !== undefined ? Boolean(current) : Boolean(setting.default)
    return createPluginBooleanGroup(ctx, isOn)
  }
  if (setting.type === 'enum' && setting.options) {
    const currentVal = current !== undefined ? String(current) : String(setting.default ?? '')
    return createPluginEnumGroup(ctx, currentVal, setting.options)
  }
  return []
}

export function createPluginSettingCommands(plugins: readonly PluginInfo[]): readonly Command[] {
  const result: Command[] = []
  for (const plugin of plugins) {
    for (const setting of plugin.settings) {
      if (!setting.palette) continue
      result.push({
        id: `plugin-setting:${plugin.id}:${setting.key}`,
        text: `${plugin.name}: ${setting.title}`,
        group: 'window',
        needs: 'choosing',
        isOffered: () => true,
      })
    }
  }
  return result
}

interface SettingsTabConfigDeps {
  dressed: ReturnType<typeof createWindowAppearance>
  sync: ReturnType<typeof syncSetting>
  nesting: ReturnType<typeof useNestingSetting>
  dayBegins: ReturnType<typeof createReviewSetting>
  rest: ReturnType<typeof createSettingsStore>
  secrets: ReturnType<typeof createSecretStore>
  file: { openSettingsFile: () => void }
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps) {
  const { dressed, sync, nesting, dayBegins, rest, secrets, file } = deps
  return {
    themes: dressed.list,
    applied: dressed.applied,
    mode: dressed.mode,
    isPinned: dressed.isPinned,
    sizes: dressed.sized,
    bounds: dressed.bounds,
    choose: (item: string) => dressed.chooseItem(item),
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
    chooseHeadings: (count: number) => nesting.chooseCount(`${count}`),
    dayStartTime: dayBegins.starts,
    latestDayStartTime: dayBegins.latest,
    chooseDayStartTime: (hour: string) => dayBegins.choose(hour),
    getSetting: (at: readonly string[]) => rest.getSetting(at),
    getModels: (at: readonly string[]) => rest.getModelsAt(at),
    write: (written: readonly SettingEdit[]) => rest.writeSettings(written),
    writeSecret: (pluginId: string, name: string, key: string) =>
      secrets.writeSecret(pluginId, name, key),
    deleteSecret: (pluginId: string, name: string) => secrets.deleteSecret(pluginId, name),
    plugins: rest.plugins,
    file: rest.path,
    openFile: () => file.openSettingsFile(),
  }
}

interface PaletteListsDeps {
  dressed: ReturnType<typeof createWindowAppearance>
  sync: ReturnType<typeof syncSetting>
  nesting: ReturnType<typeof useNestingSetting>
  rest: ReturnType<typeof createSettingsStore>
  words: Words
}

function getAppearanceStepGroups(
  command: string,
  dressed: ReturnType<typeof createWindowAppearance>,
  typed: string,
) {
  if (command === APPEARANCE) return dressed.getThemeGroups()
  if (command === MODE) return dressed.getModeGroups()
  if (command === INTERFACE_SCALE || command === TEXT_SCALE)
    return dressed.getSizeGroups(command, typed)
  return []
}

function createSettingsPaletteLists(deps: PaletteListsDeps): PaletteLists {
  const { dressed, sync, nesting, rest, words } = deps
  return {
    getStepGroups: (command, typed) => {
      const appearance = getAppearanceStepGroups(command, dressed, typed)
      if (appearance.length > 0) return appearance
      if (command === SYNCING) return sync.getSyncingGroups()
      if (command === NESTING) return nesting.getNestingGroups()
      if (command === HEADINGS) return nesting.getHeadingsGroups()
      if (command === 'performanceProfile') return getPerformanceStepGroups(words)
      if (command === 'indexingModel') return getIndexingModelStepGroups(rest, words)
      if (command.startsWith('plugin-setting:')) return getPluginStepGroups(command, rest, words)
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
  const secrets = createSecretStore(core, {
    words,
    say: log.getWriter('keys'),
    refresh: rest.start,
  })

  const file = {
    openSettingsFile: () => {
      if (rest.path.value) {
        void windowTabs.handle.openTabBeside('note', rest.path.value)
      }
    },
  }

  const configured = useSettingsTab(
    windowTabs.handle,
    createSettingsTabConfig({ dressed, sync, nesting, dayBegins, rest, secrets, file }),
  )

  watch(dressed.sized, () => onSizeChanged())

  const kept = createSettingsPaletteLists({ dressed, sync, nesting, rest, words })

  const kinds = [configured.kind]

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
