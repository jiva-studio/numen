/**
 * The settings tab: the installation it edits and the kind of tab that shows it.
 */
import type { Ref } from 'vue'
import type { CommandsHost } from '@numen/plugin-loader'
import type { createWindowAppearance } from '@/features/settings-commands'
import {
  choosePluginPath,
  createPluginRetry,
  deletePluginData,
  getPluginData,
  type SettingEdit,
  type SettingsStore,
} from '@/entities/settings'
import { listRowOptions, type ConnectionStore } from '@/entities/connection'
import { countNotes, listFiles } from '@/entities/file'
import {
  PLUGIN_FAILURE_WORDS,
  readPluginReadme,
  useCatalogEntries,
  type PluginHealthItem,
} from '@/entities/plugin'
import { runInBackground } from '@numen/ui'
import { formatError, type Caught } from '@/shared/api/errors'
import { useSettingsTab } from '../kind'
import type { Installation, InstallationPlugin, InstallationSettings } from '../types'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'

/** The health items of the plugins, and the commands of the window their fixes run. */
export interface HealthDeps {
  items: Readonly<Ref<readonly PluginHealthItem[]>>
  commands: Pick<CommandsHost, 'getIds' | 'run'>
}

export interface SettingsPageDeps {
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

function createInstallationPluginFiles() {
  return {
    getPluginData: (pluginId: string) => getPluginData(pluginId),
    deletePluginData: async (pluginId: string): Promise<string> => {
      try {
        await deletePluginData(pluginId)
        return ''
      } catch (caughtError) {
        return formatError(caughtError)
      }
    },
    getReadme: readPluginReadme,
  }
}

function createInstallationHealth({ items, commands }: HealthDeps, reportError: (error: Caught) => void) {
  return {
    items,
    canRunCommand: (command: string) => commands.getIds().includes(command),
    runCommand: (command: string, values: Readonly<Record<string, string>>) =>
      runInBackground(commands.run(command, undefined, undefined, values), reportError),
  }
}

function createInstallationPlugin(deps: InstallationPluginDeps) {
  const { rest, retry, getSettingsViews, catalogEntries, reportError } = deps
  return {
    catalog: catalogEntries,
    health: createInstallationHealth(deps.health, reportError),
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
    files: createInstallationPluginFiles(),
    plugins: rest.state.plugins,
    app: rest.state.app,
    getSettingsViews,
  }
}

function createInstallationConnections(connections: ConnectionStore) {
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
      const answer = await listRowOptions(connectionId, field, options, operations)
      return answer.ok ? answer : { ok: false as const, error: answer.error.detail }
    },
    countNotes: () => countNotes(listFiles),
  }
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps): Installation {
  return {
    settings: createInstallationSettings(deps.settings),
    plugin: createInstallationPlugin(deps.plugin),
    connections: createInstallationConnections(deps.connections),
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
}: SettingsPageDeps) {
  const getPluginName = (pluginId: string): string =>
    rest.state.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const reportError = (error: Caught) => log.getWriter('configured')(formatError(error), 'error')
  const retry = createPluginRetry({
    words: { stopped: PLUGIN_FAILURE_WORDS.stopped, retryFailed: PLUGIN_FAILURE_WORDS.retryFailed },
    notify: log.getWriter('plugins'),
    refresh: () => rest.start(),
    getName: getPluginName,
  })

  const catalogEntries = useCatalogEntries(rest.state.plugins, reportError)

  const configured = useSettingsTab(
    windowTabs.groups.handle,
    createSettingsTabConfig({
      settings: { appearance, rest, getPlatform, reportError },
      plugin: { rest, retry, getSettingsViews, catalogEntries, reportError, health },
      connections,
    }),
    reportError,
  )
  return { configured, kinds: [configured.kind] }
}
