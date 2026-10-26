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
import { type PluginInfo, type SettingEdit } from '@/entities/settings'
import { createPluginRetry, createSecretStore, createSettingsStore } from '@/entities/settings'
import { useSettingsTab } from '@/pages/settings'
import { createSettingsFileTabKind } from '@/pages/settings-file'
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
  appearance: ReturnType<typeof createWindowAppearance>
  sync: ReturnType<typeof syncSetting>
  nesting: ReturnType<typeof useNestingSetting>
  rest: ReturnType<typeof createSettingsStore>
  secrets: ReturnType<typeof createSecretStore>
  retry: ReturnType<typeof createPluginRetry>
  file: { openSettingsFile: () => void }
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps) {
  const { appearance, sync, nesting, rest, secrets, retry, file } = deps
  return {
    themes: appearance.list,
    applied: appearance.applied,
    mode: appearance.mode,
    isPinned: appearance.isPinned,
    sizes: appearance.sized,
    bounds: appearance.bounds,
    choose: (item: string) => appearance.chooseItem(item),
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
    getSetting: (targetPath: readonly string[]) => rest.getSetting(targetPath),
    getModels: (targetPath: readonly string[]) => rest.getModelsAt(targetPath),
    write: (content: readonly SettingEdit[]) => rest.writeSettings(content),
    writeSecret: (pluginId: string, name: string, key: string) =>
      secrets.writeSecret(pluginId, name, key),
    deleteSecret: (pluginId: string, name: string) => secrets.deleteSecret(pluginId, name),
    retryPlugin: (pluginId: string) => retry.retryPlugin(pluginId),
    plugins: rest.plugins,
    file: rest.path,
    openFile: () => file.openSettingsFile(),
  }
}

interface PaletteListsDeps {
  appearance: ReturnType<typeof createWindowAppearance>
  sync: ReturnType<typeof syncSetting>
  nesting: ReturnType<typeof useNestingSetting>
  rest: ReturnType<typeof createSettingsStore>
  words: Words
}

function getAppearanceStepGroups(
  command: string,
  appearance: ReturnType<typeof createWindowAppearance>,
  typed: string,
) {
  if (command === APPEARANCE) return appearance.getThemeGroups()
  if (command === MODE) return appearance.getModeGroups()
  if (command === INTERFACE_SCALE || command === TEXT_SCALE)
    return appearance.getSizeGroups(command, typed)
  return []
}

function createSettingsPaletteLists(deps: PaletteListsDeps): PaletteLists {
  const { appearance, sync, nesting, rest, words } = deps
  return {
    getStepGroups: (command, typed) => {
      const appearanceGroups = getAppearanceStepGroups(command, appearance, typed)
      if (appearanceGroups.length > 0) return appearanceGroups
      if (command === SYNCING) return sync.getSyncingGroups()
      if (command === NESTING) return nesting.getNestingGroups()
      if (command === HEADINGS) return nesting.getHeadingsGroups()
      if (command === 'performanceProfile') return getPerformanceStepGroups(words)
      if (command.startsWith('plugin-setting:')) return getPluginStepGroups(command, rest, words)
      return []
    },
    previewItem: (command, item) => {
      if (APPEARANCE_COMMANDS.includes(command)) appearance.previewItem(item)
    },
  }
}

export function useSettings({ core, words, log, windowTabs, onSizeChanged }: SettingsDeps) {
  const nesting = useNestingSetting(core, words, log.getWriter('nesting'))
  const appearance = createWindowAppearance({ core: themes, words, write: log.getWriter('worn') })
  const sync = syncSetting(core, words, log.getWriter('named'))
  const rest = createSettingsStore(core, words, log.getWriter('configured'))
  const secrets = createSecretStore(core, {
    words,
    notify: log.getWriter('keys'),
    refresh: rest.start,
  })
  const retry = createPluginRetry(core, {
    words: { stopped: words.pluginStopped, retryFailed: words.pluginRetryFailed },
    notify: log.getWriter('plugins'),
    refresh: rest.start,
    getName: (pluginId) => rest.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId,
  })

  const file = createSettingsFileTabKind(windowTabs.handle, core, () => void rest.start())

  const configured = useSettingsTab(
    windowTabs.handle,
    createSettingsTabConfig({ appearance, sync, nesting, rest, secrets, retry, file }),
  )

  watch(appearance.sized, () => onSizeChanged())

  const retainedItem = createSettingsPaletteLists({ appearance, sync, nesting, rest, words })

  const kinds = [configured.kind, file.kind]

  const start = async () => {
    void appearance.start()
    void sync.start()
    void nesting.start()
    // The plugins the window starts are named by what this read answers.
    await rest.start()
  }

  const close = () => {
    appearance.close()
  }

  const api = {
    nesting,
    appearance,
    sync,
    oneName: sync,
    rest,
    file,
    configured,
    paletteLists: retainedItem,
    kinds,
    start,
    close,
  }
  return api
}
