/**
 * Settings configuration, theme application, and palette lists for the window.
 */
import { watch } from 'vue'
import {
  APPEARANCE_WORDS,
  createWindowAppearance,
  createWindowSettingPalette,
  type SettingWords,
} from '@/features/settings-commands'
import {
  choosePluginPath,
  createPluginRetry,
  createSecretStore,
  createSettingsStore,
  SETTINGS_WORDS,
  deletePluginData,
  getPluginData,
  themes,
  watchSettings,
  type SettingEdit,
} from '@/entities/settings'
import { useCatalogEntries } from '@/entities/plugin'
import { createFollower, runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import { useSettingsTab } from '@/pages/settings'
import type { Installation } from '@/pages/settings'
import type { MessageLog } from '@/shared/notices/messages'
import { WINDOW_WORDS } from './words'
import type { WindowTabs } from '@/entities/tab'

export interface SettingsDeps {
  words: SettingWords
  log: MessageLog
  windowTabs: WindowTabs
  onSizeChanged: () => void
  getSettingsViews?: Installation['getSettingsViews']
  getPlatform?: Installation['getPlatform']
}

interface SettingsTabConfigDeps {
  appearance: ReturnType<typeof createWindowAppearance>
  rest: ReturnType<typeof createSettingsStore>
  secrets: ReturnType<typeof createSecretStore>
  retry: ReturnType<typeof createPluginRetry>
  getSettingsViews?: Installation['getSettingsViews']
  getPlatform?: Installation['getPlatform']
  catalogEntries: NonNullable<Installation['catalog']>
  reportError: (error: unknown) => void
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps) {
  const {
    appearance,
    rest,
    secrets,
    retry,
    getSettingsViews,
    getPlatform,
    catalogEntries,
    reportError,
  } = deps
  /** The window follows a write of a setting of the shell: it reads and applies the appearance. */
  const followShell = async (content: readonly SettingEdit[]): Promise<void> => {
    if (!content.some((one) => one.at[0] === 'app')) return
    await appearance.reload()
  }
  return {
    ...(getSettingsViews ? { getSettingsViews } : {}),
    ...(getPlatform ? { getPlatform } : {}),
    catalog: catalogEntries,
    getSetting: (targetPath: readonly string[]) => rest.read.getSetting(targetPath),
    getConfirmedSetting: (targetPath: readonly string[]) => rest.read.getCommittedSetting(targetPath),
    isPending: (targetPath: readonly string[]) => rest.read.isPending(targetPath),
    isRequested: (targetPath: readonly string[]) => rest.read.isRequested(targetPath),
    getModels: (targetPath: readonly string[]) => rest.read.getModelsAt(targetPath),
    write: (content: readonly SettingEdit[]) => {
      const promise = rest.write.writeSettings(content)
      runInBackground(
        promise.then(() => followShell(content)),
        reportError,
      )
      return promise
    },
    secretSections: rest.state.secretSections,
    secretMenus: rest.state.secretMenus,
    secretStates: secrets.states,
    loadSecretStates: () => secrets.loadStates(),
    credentialsPath: rest.state.credentialsPath,
    createSecret: (group: string, name: string, value: string) =>
      secrets.edit.createSecret(group, name, value),
    renameSecret: (id: string, name: string) => secrets.edit.renameSecret(id, name),
    writeSecret: (id: string, value: string) => secrets.edit.writeSecret(id, value),
    deleteSecret: (id: string) => secrets.edit.deleteSecret(id),
    deleteSecretGroup: (group: string) => secrets.edit.deleteSecretGroup(group),
    moveSecret: (id: string, beforeId: string) => secrets.edit.moveSecret(id, beforeId),
    renameSecretGroup: (group: string, name: string) => secrets.edit.renameSecretGroup(group, name),
    allowPluginSecret: (secretId: string, pluginId: string, name: string, group: string) =>
      secrets.access.allowPluginSecret(secretId, pluginId, name, group),
    denyPluginSecret: (secretId: string, pluginId: string, name: string, group: string) =>
      secrets.access.denyPluginSecret(secretId, pluginId, name, group),
    retryPlugin: (pluginId: string) => retry.retryPlugin(pluginId),
    choosePath: async (pluginId: string, key: string): Promise<string> => {
      try {
        const answer = await choosePluginPath(pluginId, key)
        if (answer.isChosen) {
          await rest.start()
        }
        return answer.problem
      } catch (caughtError) {
        return formatError(caughtError)
      }
    },
    getPluginData: (pluginId: string) => getPluginData(pluginId),
    deletePluginData: async (pluginId: string): Promise<string> => {
      try {
        await deletePluginData(pluginId)
        return ''
      } catch (caughtError) {
        return formatError(caughtError)
      }
    },
    plugins: rest.state.plugins,
    app: rest.state.app,
    file: rest.state.path,
  }
}

function createChangeFollower(deps: {
  refresh: () => Promise<unknown>
  wait?: (ms: number) => Promise<unknown>
}) {
  const { refresh, wait = sleep } = deps
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
        () => watchSettings(abortController.signal),
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
  words,
  log,
  windowTabs,
  onSizeChanged,
  getSettingsViews,
  getPlatform,
}: SettingsDeps) {
  const writeAppearanceMessage = log.getWriter('appearance')
  const writeSettingsMessage = log.getWriter('configured')
  const rest = createSettingsStore(SETTINGS_WORDS, writeSettingsMessage)
  const getPluginName = (pluginId: string): string =>
    rest.state.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const appearance = createWindowAppearance({
    core: themes,
    words: APPEARANCE_WORDS,
    write: writeAppearanceMessage,
  })
  const reportError = (error: unknown) => writeSettingsMessage(formatError(error), 'error')
  const secrets = createSecretStore({
    words: SETTINGS_WORDS,
    notify: log.getWriter('keys'),
    refresh: () => rest.start(),
  })
  const retry = createPluginRetry({
    words: { stopped: WINDOW_WORDS.pluginStopped, retryFailed: WINDOW_WORDS.pluginRetryFailed },
    notify: log.getWriter('plugins'),
    refresh: () => rest.start(),
    getName: getPluginName,
  })

  const catalogEntries = useCatalogEntries(rest.state.plugins, reportError)

  const configured = useSettingsTab(
    windowTabs.groups.handle,
    createSettingsTabConfig({
      appearance,
      rest,
      secrets,
      retry,
      ...(getSettingsViews ? { getSettingsViews } : {}),
      ...(getPlatform ? { getPlatform } : {}),
      catalogEntries,
      reportError,
    }),
    reportError,
  )

  watch(appearance.sized, () => onSizeChanged())

  const settingPalette = createWindowSettingPalette({ rest, appearance, words })

  const changes = createChangeFollower({
    refresh: () => Promise.all([rest.start(), appearance.reload()]),
  })

  const start = async () => {
    runInBackground(appearance.start(), reportError)
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
    rest,
    configured,
    paletteLists: settingPalette.lists,
    settingPalette,
    kinds: [configured.kind],
    start,
    close,
  }
  return api
}
