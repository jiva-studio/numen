/**
 * Settings configuration, theme dressing, and palette lists for the window.
 */
import { computed, watch } from 'vue'
import { themes } from '@/entities/settings'
import {
  APPEARANCE,
  APPEARANCE_COMMANDS,
  INTERFACE_SCALE,
  MODE,
  OFF,
  ON,
  SYNCING,
  syncSetting,
  TEXT_SCALE,
  createWindowAppearance,
} from '@/features/settings-commands'
import {
  formatValue,
  getOptions,
  getValue,
  type PluginInfo,
  type PluginSettingOption,
  type SettingEdit,
} from '@/entities/settings'
import { formatErrorMessage, type PluginCatalog } from '@numen/plugin-sdk'
import { useCatalogEntries } from '@/entities/plugin'
import { runInBackground } from '@numen/ui'
import { createPluginRetry, createSecretStore, createSettingsStore } from '@/entities/settings'
import { useSettingsTab } from '@/pages/settings'
import type { Installation } from '@/pages/settings'
import type { Command, PaletteLists } from '@/features/command-palette'
import type { SettingsPort } from '@/app/ports/settings'
import { getNamedIcon } from '@/shared/icons'
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
  getSettingsViews?: Installation['getSettingsViews']
  /** The catalog the plugin list reads; without it the list shows the installed plugins only. */
  catalog?: PluginCatalog
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
  options: readonly PluginSettingOption[],
) {
  return [
    {
      id: ctx.id,
      title: ctx.title,
      items: options.map((opt) => ({
        id: opt.value,
        title: opt.title || opt.value,
        ...(opt.value === current ? { detail: ctx.words.current, isCurrent: true } : {}),
      })),
    },
  ]
}

function findPluginSetting(plugins: readonly PluginInfo[], command: string) {
  const [, pluginId, key] = command.split(':')
  if (!pluginId || !key) return null
  const plugin = plugins.find((p) => p.id === pluginId)
  const setting = plugin?.settings.find((s) => s.key === key)
  if (!plugin || !setting) return null
  return { plugin, pluginId, key, setting }
}

function getPluginStepGroups(
  command: string,
  rest: ReturnType<typeof createSettingsStore>,
  words: Words,
) {
  const match = findPluginSetting(rest.plugins.value, command)
  if (!match) return []
  const { plugin, pluginId, key, setting } = match
  const read = (one: string) => rest.getSetting(['plugins', pluginId, one])
  const current = read(key)
  const ctx: PluginGroupContext = { id: `plugin-${pluginId}-${key}`, title: setting.title, words }

  if (setting.type === 'boolean') {
    const isOn = current !== undefined ? Boolean(current) : Boolean(setting.default)
    return createPluginBooleanGroup(ctx, isOn)
  }
  if (setting.type === 'enum') {
    const currentVal = formatValue(getValue(plugin, setting, read).value)
    return createPluginEnumGroup(ctx, currentVal, getOptions(plugin, setting, read))
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
        group: 'settings',
        needs: 'choosing',
        ...(setting.icon ? { icon: getNamedIcon(setting.icon) } : {}),
        isAvailable: () => true,
      })
    }
  }
  return result
}

interface SettingsTabConfigDeps {
  appearance: ReturnType<typeof createWindowAppearance>
  sync: ReturnType<typeof syncSetting>
  rest: ReturnType<typeof createSettingsStore>
  secrets: ReturnType<typeof createSecretStore>
  retry: ReturnType<typeof createPluginRetry>
  core: SettingsPort
  scheduleRefresh: () => void
  getSettingsViews?: Installation['getSettingsViews']
  catalogEntries?: Installation['catalog']
  reportError: (error: unknown) => void
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps) {
  const {
    appearance,
    sync,
    rest,
    secrets,
    retry,
    core,
    scheduleRefresh,
    getSettingsViews,
    catalogEntries,
    reportError,
  } = deps
  return {
    ...(getSettingsViews ? { getSettingsViews } : {}),
    ...(catalogEntries ? { catalog: catalogEntries } : {}),
    themes: appearance.list,
    applied: appearance.applied,
    mode: appearance.mode,
    isPinned: appearance.isPinned,
    sizes: appearance.sized,
    bounds: appearance.bounds,
    choose: (item: string) => appearance.chooseItem(item),
    isSyncing: computed({
      get: () => sync.isEnabled.value,
      set: (on: boolean) => runInBackground(sync.choose(on ? ON : OFF), reportError),
    }),
    getSetting: (targetPath: readonly string[]) => rest.getSetting(targetPath),
    getModels: (targetPath: readonly string[]) => rest.getModelsAt(targetPath),
    write: (content: readonly SettingEdit[]) => {
      const promise = rest.writeSettings(content)
      runInBackground(promise.then(scheduleRefresh), reportError)
      return promise
    },
    writeSecret: (pluginId: string, name: string, key: string) =>
      secrets.writeSecret(pluginId, name, key),
    deleteSecret: (pluginId: string, name: string) => secrets.deleteSecret(pluginId, name),
    retryPlugin: (pluginId: string) => retry.retryPlugin(pluginId),
    choosePath: async (pluginId: string, key: string): Promise<string> => {
      try {
        const answer = await core.choosePluginPath(pluginId, key)
        if (answer.isChosen) {
          await rest.start()
          scheduleRefresh()
        }
        return answer.problem
      } catch (caughtError) {
        return formatErrorMessage(caughtError)
      }
    },
    getPluginData: (pluginId: string) => core.getPluginData(pluginId),
    deletePluginData: async (pluginId: string): Promise<string> => {
      try {
        await core.deletePluginData(pluginId)
        return ''
      } catch (caughtError) {
        return formatErrorMessage(caughtError)
      }
    },
    plugins: rest.plugins,
    file: rest.path,
  }
}

interface PaletteListsDeps {
  appearance: ReturnType<typeof createWindowAppearance>
  sync: ReturnType<typeof syncSetting>
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
  const { appearance, sync, rest, words } = deps
  return {
    getStepGroups: (command, typed) => {
      const appearanceGroups = getAppearanceStepGroups(command, appearance, typed)
      if (appearanceGroups.length > 0) return appearanceGroups
      if (command === SYNCING) return sync.getSyncingGroups()
      if (command === 'performanceProfile') return getPerformanceStepGroups(words)
      if (command.startsWith('plugin-setting:')) return getPluginStepGroups(command, rest, words)
      return []
    },
    previewItem: (command, item) => {
      if (APPEARANCE_COMMANDS.includes(command)) appearance.previewItem(item)
    },
  }
}

export function useSettings({
  core,
  words,
  log,
  windowTabs,
  onSizeChanged,
  getSettingsViews,
  catalog,
}: SettingsDeps) {
  const appearance = createWindowAppearance({ core: themes, words, write: log.getWriter('worn') })
  const sync = syncSetting(core, words, log.getWriter('named'))
  const writeSettingsMessage = log.getWriter('configured')
  const rest = createSettingsStore(core, words, writeSettingsMessage)
  const refreshTimers: ReturnType<typeof setTimeout>[] = []
  const reportError = (error: unknown) => writeSettingsMessage(formatErrorMessage(error), 'error')

  const clearRefreshTimers = () => {
    while (refreshTimers.length > 0) {
      clearTimeout(refreshTimers.pop())
    }
  }

  const scheduleRefresh = () => {
    clearRefreshTimers()
    for (const delayMs of [1500, 3500, 7000, 11000, 16000]) {
      refreshTimers.push(
        setTimeout(() => {
          runInBackground(rest.start(), reportError)
        }, delayMs),
      )
    }
  }

  const refreshAndSchedule = async () => {
    await rest.start()
    scheduleRefresh()
  }

  const secrets = createSecretStore(core, {
    words,
    notify: log.getWriter('keys'),
    refresh: refreshAndSchedule,
  })
  const retry = createPluginRetry(core, {
    words: { stopped: words.pluginStopped, retryFailed: words.pluginRetryFailed },
    notify: log.getWriter('plugins'),
    refresh: refreshAndSchedule,
    getName: (pluginId) => rest.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId,
  })

  const catalogEntries = catalog ? useCatalogEntries(catalog, rest.plugins, reportError) : undefined

  const configured = useSettingsTab(
    windowTabs.handle,
    createSettingsTabConfig({
      appearance,
      sync,
      rest,
      secrets,
      retry,
      core,
      scheduleRefresh,
      ...(getSettingsViews ? { getSettingsViews } : {}),
      ...(catalogEntries ? { catalogEntries } : {}),
      reportError,
    }),
    reportError,
  )

  watch(appearance.sized, () => onSizeChanged())

  const retainedItem = createSettingsPaletteLists({ appearance, sync, rest, words })

  const kinds = [configured.kind]

  const start = async () => {
    runInBackground(appearance.start(), reportError)
    runInBackground(sync.start(), reportError)
    // The plugins the window starts are named by what this read answers.
    await rest.start()
    scheduleRefresh()
  }

  const close = () => {
    clearRefreshTimers()
    appearance.close()
  }

  const api = {
    appearance,
    sync,
    oneName: sync,
    rest,
    configured,
    paletteLists: retainedItem,
    kinds,
    start,
    close,
  }
  return api
}
