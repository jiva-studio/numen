/**
 * Settings configuration, theme application, and palette lists for the window.
 */
import { watch } from 'vue'
import { themes } from '@/entities/settings'
import { INTERFACE_SCALE, TEXT_SCALE, createWindowAppearance } from '@/features/settings-commands'
import { APP_OWNER, createSettingPalette, type SettingPreview } from './settingCommands'
import { MODES, type Mode, type SettingEdit } from '@/entities/settings'
import {
  formatErrorMessage,
  type ConnectionClient,
  type ModelClient,
  type PluginCatalog,
} from '@numen/core-client'
import { useCatalogEntries } from '@/entities/plugin'
import { createConnectionStore } from '@/entities/connection'
import { createFollower, runInBackground } from '@numen/ui'
import { createPluginRetry, createSecretStore, createSettingsStore } from '@/entities/settings'
import { useSettingsTab } from '@/pages/settings'
import type { Installation } from '@/pages/settings'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import type { WORDS } from '@/shared/words'
import type { useWindowTabs } from '@/entities/tab'

type Words = typeof WORDS

export interface SettingsDeps {
  core: SettingsPort
  words: Words
  log: MessageLog
  windowTabs: ReturnType<typeof useWindowTabs>
  onSizeChanged: () => void
  getSettingsViews?: Installation['getSettingsViews']
  getPlatform?: Installation['getPlatform']
  catalog?: PluginCatalog
  connections?: ConnectionClient
  models?: ModelClient
}

interface SettingsTabConfigDeps {
  models?: ModelClient
  appearance: ReturnType<typeof createWindowAppearance>
  rest: ReturnType<typeof createSettingsStore>
  secrets: ReturnType<typeof createSecretStore>
  connections?: ReturnType<typeof createConnectionStore>
  retry: ReturnType<typeof createPluginRetry>
  core: SettingsPort
  getSettingsViews?: Installation['getSettingsViews']
  getPlatform?: Installation['getPlatform']
  catalogEntries?: Installation['catalog']
  reportError: (error: unknown) => void
}

function createConnectionMembers(
  connections: ReturnType<typeof createConnectionStore> | undefined,
): Partial<Installation> {
  if (!connections) return {}
  return {
    connections: connections.connections,
    connectionTypes: connections.types,
    loadConnections: () => connections.load(),
    createConnection: connections.create,
    updateConnection: connections.update,
    deleteConnection: connections.delete,
  }
}

function createConnections(client: ConnectionClient | undefined, words: Words, log: MessageLog) {
  if (!client) return undefined
  return createConnectionStore(client, {
    describeFailure: words.connectionFailure,
    loadFailed: words.connectionsLoadFailed,
    notify: log.getWriter('connections'),
  })
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps) {
  const {
    appearance,
    rest,
    secrets,
    connections,
    retry,
    core,
    getSettingsViews,
    getPlatform,
    catalogEntries,
    reportError,
    models,
  } = deps
  /** The window follows a write of a setting of the shell: it reads and applies the appearance. */
  const followShell = async (content: readonly SettingEdit[]): Promise<void> => {
    if (!content.some((one) => one.at[0] === 'app')) return
    await appearance.reload()
  }
  return {
    ...(getSettingsViews ? { getSettingsViews } : {}),
    ...(getPlatform ? { getPlatform } : {}),
    ...(catalogEntries ? { catalog: catalogEntries } : {}),
    ...createConnectionMembers(connections),
    ...(models
      ? {
          listModels: (connectionId: string, operation: string) =>
            models.listModels(connectionId, operation),
          listReasoningLevels: (connectionId: string, model: string) =>
            models.listReasoningLevels(connectionId, model),
        }
      : {}),
    getSetting: (targetPath: readonly string[]) => rest.getSetting(targetPath),
    getConfirmedSetting: (targetPath: readonly string[]) => rest.getCommittedSetting(targetPath),
    isPending: (targetPath: readonly string[]) => rest.isPending(targetPath),
    isRequested: (targetPath: readonly string[]) => rest.isRequested(targetPath),
    getModels: (targetPath: readonly string[]) => rest.getModelsAt(targetPath),
    write: (content: readonly SettingEdit[]) => {
      const promise = rest.writeSettings(content)
      runInBackground(
        promise.then(() => followShell(content)),
        reportError,
      )
      return promise
    },
    secretSections: rest.secretSections,
    secretMenus: rest.secretMenus,
    secretStates: secrets.states,
    loadSecretStates: () => secrets.loadStates(),
    credentialsPath: rest.credentialsPath,
    createSecret: (group: string, name: string, value: string) =>
      secrets.createSecret(group, name, value),
    renameSecret: (id: string, name: string) => secrets.renameSecret(id, name),
    writeSecret: (id: string, value: string) => secrets.writeSecret(id, value),
    deleteSecret: (id: string) => secrets.deleteSecret(id),
    deleteSecretGroup: (group: string) => secrets.deleteSecretGroup(group),
    moveSecret: (id: string, beforeId: string) => secrets.moveSecret(id, beforeId),
    renameSecretGroup: (group: string, name: string) => secrets.renameSecretGroup(group, name),
    allowPluginSecret: (secretId: string, pluginId: string, name: string, group: string) =>
      secrets.allowPluginSecret(secretId, pluginId, name, group),
    denyPluginSecret: (secretId: string, pluginId: string, name: string, group: string) =>
      secrets.denyPluginSecret(secretId, pluginId, name, group),
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

const SHELL_APPEARANCE = ['app', 'appearance'] as const

function createWindowPreview(
  appearance: ReturnType<typeof createWindowAppearance>,
): SettingPreview {
  return {
    apply: (path, value) => {
      if (path.length !== 3 || !SHELL_APPEARANCE.every((one, index) => path[index] === one)) return
      const key = path[2]
      if (key === 'theme' && typeof value === 'string') appearance.previewTheme(value)
      if (key === 'mode' && isMode(value)) appearance.previewMode(value)
      if (key === INTERFACE_SCALE && typeof value === 'number') {
        appearance.previewSize(INTERFACE_SCALE, value)
      }
      if (key === TEXT_SCALE && typeof value === 'number') {
        appearance.previewSize(TEXT_SCALE, value)
      }
    },
    restore: () => appearance.restore(),
  }
}

const isMode = (value: unknown): value is Mode => MODES.some((one) => one === value)

function createWindowSettingPalette(deps: {
  rest: ReturnType<typeof createSettingsStore>
  appearance: ReturnType<typeof createWindowAppearance>
  words: Words
}) {
  const { rest, appearance, words } = deps
  return createSettingPalette({
    getOwners: () => [
      { id: APP_OWNER, name: '', settings: rest.app.value },
      ...rest.plugins.value.map((one) => ({ id: one.id, name: one.name, settings: one.settings })),
    ],
    words,
    preview: createWindowPreview(appearance),
    write: async (edits) => {
      await rest.writeSettings(edits)
      // The window shows what the settings now hold, also where the write was not accepted.
      if (edits.some((one) => one.at[0] === APP_OWNER && one.at[1] === 'appearance')) {
        await appearance.reload()
      }
    },
  })
}

export function useSettings({
  core,
  words,
  log,
  windowTabs,
  onSizeChanged,
  getSettingsViews,
  getPlatform,
  catalog,
  connections: connectionClient,
  models,
}: SettingsDeps) {
  const writeAppearanceMessage = log.getWriter('appearance')
  const writeSettingsMessage = log.getWriter('configured')
  const rest = createSettingsStore(core, words, writeSettingsMessage)
  const getPluginName = (pluginId: string): string =>
    rest.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const appearance = createWindowAppearance({
    core: themes,
    words,
    write: writeAppearanceMessage,
  })
  const reportError = (error: unknown) => writeSettingsMessage(formatErrorMessage(error), 'error')
  const secrets = createSecretStore(core, {
    words,
    notify: log.getWriter('keys'),
    refresh: () => rest.start(),
  })
  const connections = createConnections(connectionClient, words, log)
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
      rest,
      secrets,
      ...(connections ? { connections } : {}),
      retry,
      core,
      ...(getSettingsViews ? { getSettingsViews } : {}),
      ...(getPlatform ? { getPlatform } : {}),
      ...(catalogEntries ? { catalogEntries } : {}),
      reportError,
      ...(models ? { models } : {}),
    }),
    reportError,
  )

  watch(appearance.sized, () => onSizeChanged())

  const settingPalette = createWindowSettingPalette({ rest, appearance, words })

  const changes = createChangeFollower({
    core,
    refresh: () => Promise.all([rest.start(), appearance.reload(), connections?.reload()]),
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
