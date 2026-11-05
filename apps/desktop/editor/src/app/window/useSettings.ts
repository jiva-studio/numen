/**
 * Settings configuration, theme application, and palette lists for the window.
 */
import { watch } from 'vue'
import { themes } from '@/entities/settings'
import {
  APPEARANCE_COMMANDS,
  syncSetting,
  createWindowAppearance,
} from '@/features/settings-commands'
import { COMMAND } from '@/shared/commandIds'
import { type PluginInfo, type PluginSettingOption, type SettingEdit } from '@/entities/settings'
import { formatErrorMessage, type PluginCatalog } from '@numen/core-client'
import { useCatalogEntries } from '@/entities/plugin'
import { createFollower, runInBackground } from '@numen/ui'
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
  const plugin = plugins.find((p) => p.id === pluginId)
  if (!plugin) return null
  const setting = plugin.settings.find((s) => s.key === key)
  if (!setting) return null
  return { plugin, pluginId: plugin.id, key: setting.key, setting }
}

function getPluginStepGroups(
  command: string,
  rest: ReturnType<typeof createSettingsStore>,
  words: Words,
) {
  const match = findPluginSetting(rest.plugins.value, command)
  if (!match) return []
  const { pluginId, key, setting } = match
  const current = rest.getSetting([pluginId, key])
  const ctx: PluginGroupContext = { id: `plugin-${pluginId}-${key}`, title: setting.title, words }

  if (setting.type === 'boolean') {
    const isOn = Boolean(current)
    return createPluginBooleanGroup(ctx, isOn)
  }
  if (setting.type === 'enum') {
    const resolved = setting.value ?? current ?? setting.default
    const currentVal = typeof resolved === 'string' ? resolved : JSON.stringify(resolved ?? '')
    return createPluginEnumGroup(ctx, currentVal, setting.activeOptions ?? setting.options ?? [])
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
        input: 'choice',
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
    getSettingsViews,
    catalogEntries,
    reportError,
  } = deps
  /** The window follows a write of a setting of the shell: it reads and applies the appearance. */
  const followShell = async (content: readonly SettingEdit[]): Promise<void> => {
    if (!content.some((one) => one.at[0] === 'app')) return
    await Promise.all([appearance.reload(), sync.start()])
  }
  return {
    ...(getSettingsViews ? { getSettingsViews } : {}),
    ...(catalogEntries ? { catalog: catalogEntries } : {}),
    getSetting: (targetPath: readonly string[]) => rest.getSetting(targetPath),
    getConfirmedSetting: (targetPath: readonly string[]) => rest.getCommittedSetting(targetPath),
    isPending: (targetPath: readonly string[]) => rest.isPending(targetPath),
    getModels: (targetPath: readonly string[]) => rest.getModelsAt(targetPath),
    write: (content: readonly SettingEdit[]) => {
      const promise = rest.writeSettings(content)
      runInBackground(promise.then(() => followShell(content)), reportError)
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
    app: rest.app,
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
  if (command === COMMAND.chooseTheme) return appearance.getThemeGroups()
  if (command === COMMAND.chooseMode) return appearance.getModeGroups()
  if (command === COMMAND.chooseInterfaceSize || command === COMMAND.chooseTextSize)
    return appearance.getSizeGroups(command, typed)
  return []
}

function createSettingsPaletteLists(deps: PaletteListsDeps): PaletteLists {
  const { appearance, sync, rest, words } = deps
  return {
    getStepGroups: (command, typed) => {
      const appearanceGroups = getAppearanceStepGroups(command, appearance, typed)
      if (appearanceGroups.length > 0) return appearanceGroups
      if (command === COMMAND.chooseTitleSync) return sync.getSyncingGroups()
      if (command === COMMAND.choosePerformanceProfile) return getPerformanceStepGroups(words)
      if (command.startsWith('plugin-setting:')) return getPluginStepGroups(command, rest, words)
      return []
    },
    previewItem: (command, item) => {
      if (APPEARANCE_COMMANDS.includes(command)) appearance.previewItem(item)
    },
  }
}

/**
 * Follows the settings that change, whoever changes them: this window, another window, a plugin, the
 * palette, or a person editing the file. Each change is read again once.
 */
function createChangeFollower(deps: {
  core: SettingsPort
  refresh: () => Promise<unknown>
  wait?: (ms: number) => Promise<unknown>
}) {
  const { core, refresh, wait = sleep } = deps
  const abortController = new AbortController()
  let isOpen = true
  // The stream opens with an empty batch. After the stream was lost, what changed meanwhile is read
  // with it.
  let isResumed = false
  const follows = createFollower({
    isOpen: () => isOpen,
    // The stream of the themes reports the lost connection.
    setLost: () => undefined,
    wait,
    reset: () => {
      isResumed = true
    },
  })
  return {
    start: () =>
      follows(
        () => core.watchSettings(abortController.signal),
        async (batch) => {
          if (batch.length === 0 && !isResumed) return
          isResumed = false
          await refresh()
        },
      ),
    close: () => {
      isOpen = false
      abortController.abort()
    },
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export function useSettings({
  core,
  words,
  log,
  windowTabs,
  onSizeChanged,
  getSettingsViews,
  catalog,
}: SettingsDeps) {
  const writeAppearanceMessage = log.getWriter('appearance')
  const sync = syncSetting(core, words, log.getWriter('named'))
  const writeSettingsMessage = log.getWriter('configured')
  const rest = createSettingsStore(core, words, writeSettingsMessage)
  const getPluginName = (pluginId: string): string =>
    rest.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const appearance = createWindowAppearance({
    core: themes,
    words,
    write: writeAppearanceMessage,
    getPluginName,
  })
  const reportError = (error: unknown) => writeSettingsMessage(formatErrorMessage(error), 'error')
  const secrets = createSecretStore(core, {
    words,
    notify: log.getWriter('keys'),
    refresh: () => rest.start(),
  })
  const retry = createPluginRetry(core, {
    words: { stopped: words.pluginStopped, retryFailed: words.pluginRetryFailed },
    notify: log.getWriter('plugins'),
    refresh: () => rest.start(),
    getName: getPluginName,
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
      ...(getSettingsViews ? { getSettingsViews } : {}),
      ...(catalogEntries ? { catalogEntries } : {}),
      reportError,
    }),
    reportError,
  )

  watch(appearance.sized, () => onSizeChanged())

  const retainedItem = createSettingsPaletteLists({ appearance, sync, rest, words })

  const kinds = [configured.kind]

  const changes = createChangeFollower({
    core,
    refresh: () => Promise.all([rest.start(), appearance.reload(), sync.start()]),
  })

  const start = async () => {
    runInBackground(appearance.start(), reportError)
    runInBackground(sync.start(), reportError)
    runInBackground(changes.start(), reportError)
    // The plugins the window starts are named by what this read answers.
    await rest.start()
  }

  const close = () => {
    changes.close()
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
