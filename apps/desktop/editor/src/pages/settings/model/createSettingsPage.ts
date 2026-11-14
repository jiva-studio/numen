/**
 * The settings tab: the installation it edits and the kind of tab that shows it.
 */
import type { Ref } from 'vue'
import type { CommandsHost } from '@numen/plugin-loader'
import type { createWindowAppearance } from '@/features/settings-commands'
import {
  createPluginRetry,
  type SettingEdit,
  type SettingsApi,
  type SettingsStore,
} from '@/entities/settings'
import type { ConnectionStore } from '@/entities/connection'
import { countNotes, type FileApi } from '@/entities/file'
import {
  PLUGIN_FAILURE_WORDS,
  readPluginReadme,
  useCatalogEntries,
  type CatalogReader,
  type PluginHealthItem,
} from '@/entities/plugin'
import { runInBackground } from '@numen/ui'
import { formatError, type Caught } from '@/shared/api/errors'
import { useSettingsTab } from '../kind'
import type { Installation, InstallationPlugin, InstallationSettings } from '../types'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'

/** The health items of the plugins, and the commands of the window their fixes run. */
interface HealthDeps {
  items: Readonly<Ref<readonly PluginHealthItem[]>>
  commands: Pick<CommandsHost, 'getIds' | 'run'>
}

/** The calls of the core the settings tab makes. */
export interface SettingsPageApi {
  readonly settings: Pick<
    SettingsApi,
    'choosePluginPath' | 'getPluginData' | 'deletePluginData' | 'retryPlugin'
  >
  readonly listFiles: FileApi['listFiles']
  readonly listCatalog: CatalogReader
}

export interface SettingsPageDeps {
  api: SettingsPageApi
  rest: SettingsStore
  connections: ConnectionStore
  appearance: ReturnType<typeof createWindowAppearance>
  log: MessageLog
  windowTabs: WindowTabs
  getSettingsViews: InstallationPlugin['getSettingsViews']
  getPlatform: InstallationSettings['getPlatform']
  health: HealthDeps
}

interface InstallationSettingsDeps {
  appearance: ReturnType<typeof createWindowAppearance>
  rest: SettingsStore
  getPlatform: InstallationSettings['getPlatform']
  reportError: (error: Caught) => void
}

interface InstallationPluginDeps {
  settings: SettingsPageApi['settings']
  rest: SettingsStore
  retry: ReturnType<typeof createPluginRetry>
  getSettingsViews: InstallationPlugin['getSettingsViews']
  catalogEntries: NonNullable<InstallationPlugin['catalog']>
  reportError: (error: Caught) => void
  health: HealthDeps
}

interface SettingsTabConfigDeps {
  settings: InstallationSettingsDeps
  plugin: InstallationPluginDeps
  connections: ConnectionStore
  listFiles: SettingsPageApi['listFiles']
}

function createInstallationSettings(deps: InstallationSettingsDeps) {
  const { appearance, rest, getPlatform, reportError } = deps
  /** The window follows a write of a setting of the shell: it reads and applies the appearance. */
  const followShell = async (content: readonly SettingEdit[]): Promise<void> => {
    if (!content.some((one) => one.at[0] === 'app')) return
    await appearance.reload()
  }
  return {
    getSetting: (targetPath: readonly string[]) => rest.read.getSetting(targetPath),
    getConfirmedSetting: (targetPath: readonly string[]) =>
      rest.read.getCommittedSetting(targetPath),
    isPending: (targetPath: readonly string[]) => rest.read.isPending(targetPath),
    isRequested: (targetPath: readonly string[]) => rest.read.isRequested(targetPath),
    write: (content: readonly SettingEdit[]) => {
      const promise = rest.write.writeSettings(content)
      runInBackground(
        promise.then(() => followShell(content)),
        reportError,
      )
      return promise
    },
    file: rest.state.path,
    getPlatform,
  }
}

function createInstallationPluginFiles(settings: SettingsPageApi['settings']) {
  return {
    getPluginData: (pluginId: string) => settings.getPluginData(pluginId),
    deletePluginData: async (pluginId: string): Promise<string> => {
      try {
        await settings.deletePluginData(pluginId)
        return ''
      } catch (caughtError) {
        return formatError(caughtError)
      }
    },
    getReadme: readPluginReadme,
  }
}

function createInstallationHealth(
  { items, commands }: HealthDeps,
  reportError: (error: Caught) => void,
) {
  return {
    items,
    canRunCommand: (command: string) => commands.getIds().includes(command),
    runCommand: (command: string, values: Readonly<Record<string, string>>) =>
      runInBackground(commands.run(command, undefined, undefined, values), reportError),
  }
}

function createInstallationPlugin(deps: InstallationPluginDeps) {
  const { settings, rest, retry, getSettingsViews, catalogEntries, reportError } = deps
  return {
    catalog: catalogEntries,
    health: createInstallationHealth(deps.health, reportError),
    retryPlugin: (pluginId: string) => retry.retryPlugin(pluginId),
    choosePath: async (pluginId: string, key: string): Promise<string> => {
      try {
        const answer = await settings.choosePluginPath(pluginId, key)
        if (answer.isChosen) {
          await rest.start()
        }
        return answer.problem
      } catch (caughtError) {
        return formatError(caughtError)
      }
    },
    files: createInstallationPluginFiles(settings),
    plugins: rest.state.plugins,
    app: rest.state.app,
    getSettingsViews,
  }
}

function createInstallationConnections(
  connections: ConnectionStore,
  listFiles: SettingsPageApi['listFiles'],
) {
  return {
    list: connections.connections,
    types: connections.types,
    operationTitles: connections.operationTitles,
    load: () => connections.load(),
    edits: {
      create: connections.create,
      update: connections.update,
      delete: connections.delete,
    },
    listRowOptions: async (
      connectionId: string,
      field: string,
      options: Readonly<Record<string, string>>,
      operations: readonly string[],
    ) => {
      const answer = await connections.listRowOptions(connectionId, field, options, operations)
      return answer.ok ? answer : { ok: false as const, error: answer.error.detail }
    },
    countNotes: () => countNotes(listFiles),
  }
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps): Installation {
  return {
    settings: createInstallationSettings(deps.settings),
    plugin: createInstallationPlugin(deps.plugin),
    connections: createInstallationConnections(deps.connections, deps.listFiles),
  }
}

/** The settings tab of a window: the page that edits numen.json, as a kind of tab. */
export function createSettingsPage({
  rest,
  connections,
  appearance,
  log,
  windowTabs,
  getSettingsViews,
  getPlatform,
  health,
  api,
}: SettingsPageDeps) {
  const getPluginName = (pluginId: string): string =>
    rest.state.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const reportError = (error: Caught) => log.getWriter('configured')(formatError(error), 'error')
  const retry = createPluginRetry({
    askPluginRetry: api.settings.retryPlugin,
    words: { stopped: PLUGIN_FAILURE_WORDS.stopped, retryFailed: PLUGIN_FAILURE_WORDS.retryFailed },
    notify: log.getWriter('plugins'),
    refresh: () => rest.start(),
    getName: getPluginName,
  })

  const catalogEntries = useCatalogEntries(api.listCatalog, rest.state.plugins, reportError)

  const configured = useSettingsTab(
    windowTabs.groups.handle,
    createSettingsTabConfig({
      settings: { appearance, rest, getPlatform, reportError },
      plugin: {
        settings: api.settings,
        rest,
        retry,
        getSettingsViews,
        catalogEntries,
        reportError,
        health,
      },
      connections,
      listFiles: api.listFiles,
    }),
    reportError,
  )
  return { configured, kinds: [configured.kind] }
}
