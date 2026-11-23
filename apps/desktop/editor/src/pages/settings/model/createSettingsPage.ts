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
import type { ConnectionApi, ConnectionStore } from '@/entities/connection'
import {
  createIdentityStore,
  type IdentityApi,
  type IdentityStore,
  type KitProof,
} from '@/entities/identity'
import { createLicenseStore, type LicenseApi, type LicenseStore } from '@/entities/license'
import { countNotes, type FileApi } from '@/entities/file'
import {
  PLUGIN_FAILURE_WORDS,
  useCatalogEntries,
  type CatalogReader,
  type ReadmeReader,
  type PluginHealthItem,
} from '@/entities/plugin'
import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import { createPrinter } from '@/pages/settings/api/print'
import { getIdentityViews } from '../lib/settingsViews'
import { useSettingsTabKind } from '../useSettingsTabKind'
import type { Installation, InstallationPlugin, InstallationSettings } from '../types'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'

/** The health items of the plugins, and the commands of the window their fixes run. */
interface HealthDeps {
  items: Readonly<Ref<readonly PluginHealthItem[]>>
  commands: Pick<CommandsHost, 'getIds' | 'run'>
}

/** The calls of the core the settings tab makes. */
interface SettingsPageApi {
  readonly settings: Pick<
    SettingsApi,
    'choosePluginPath' | 'getPluginData' | 'deletePluginData' | 'retryPlugin'
  >
  readonly license: LicenseApi
  readonly identity: IdentityApi
  readonly listFiles: FileApi['listFiles']
  readonly listCatalog: CatalogReader
  readonly readReadme: ReadmeReader
  readonly listRowOptions: ConnectionApi['listRowOptions']
}

/** The slots a plugin draws, and the plugin each entry of them comes from. */
type SlotGroupsReader = (pluginId: string) => readonly {
  readonly slot: string
  readonly title: string
  readonly contributors: readonly {
    readonly key: string
    readonly label: string
    readonly pluginId: string
    readonly group?: string | undefined
  }[]
}[]

/** What the settings tab reads of the plugins: the views they draw in it, the slots, and their health. */
interface SettingsPagePlugins {
  readonly getSettingsViews: InstallationPlugin['getSettingsViews']
  readonly getSlotGroups?: SlotGroupsReader
  readonly health: HealthDeps
}

export interface SettingsPageDeps {
  api: SettingsPageApi
  rest: SettingsStore
  connections: ConnectionStore
  appearance: ReturnType<typeof createWindowAppearance>
  log: MessageLog
  windowTabs: WindowTabs
  getPlatform: InstallationSettings['getPlatform']
  plugins: SettingsPagePlugins
}

interface InstallationSettingsDeps {
  appearance: ReturnType<typeof createWindowAppearance>
  rest: SettingsStore
  getPlatform: InstallationSettings['getPlatform']
  reportError: (error: unknown) => void
}

interface InstallationPluginDeps {
  api: Pick<SettingsPageApi, 'settings' | 'readReadme'>
  rest: SettingsStore
  retry: ReturnType<typeof createPluginRetry>
  getSettingsViews: InstallationPlugin['getSettingsViews']
  getSlotGroups?: SlotGroupsReader
  catalogEntries: NonNullable<InstallationPlugin['catalog']>
  reportError: (error: unknown) => void
  health: HealthDeps
}

interface SettingsTabConfigDeps {
  settings: InstallationSettingsDeps
  plugin: InstallationPluginDeps
  connections: ConnectionStore
  licenses: LicenseStore
  identity: IdentityStore
  api: Pick<SettingsPageApi, 'listFiles' | 'listRowOptions'>
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

function createInstallationPluginFiles(
  settings: SettingsPageApi['settings'],
  readReadme: ReadmeReader,
) {
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
    getReadme: readReadme,
  }
}

function createInstallationHealth(
  { items, commands }: HealthDeps,
  reportError: (error: unknown) => void,
) {
  return {
    items,
    canRunCommand: (command: string) => commands.getIds().includes(command),
    runCommand: (command: string, values: Readonly<Record<string, string>>) =>
      runInBackground(commands.run(command, undefined, undefined, values), reportError),
  }
}

function createInstallationSlots(deps: Pick<InstallationPluginDeps, 'rest' | 'getSlotGroups'>) {
  const { rest, getSlotGroups } = deps
  if (!getSlotGroups) return {}
  const getPluginName = (pluginId: string): string =>
    pluginId === 'app'
      ? 'Numen'
      : (rest.state.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId)
  return {
    getSlotGroups: (pluginId: string) =>
      getSlotGroups(pluginId).map((group) => ({
        ...group,
        contributors: group.contributors.map((one) => ({
          key: one.key,
          label: one.label,
          group: one.group,
          pluginName: getPluginName(one.pluginId),
        })),
      })),
  }
}

function createInstallationDeclared(deps: Pick<InstallationPluginDeps, 'rest'>) {
  const { app, appGroups } = deps.rest.state
  return { app, appGroups }
}

function createInstallationPlugin(deps: InstallationPluginDeps) {
  const { api, rest, retry, getSettingsViews, catalogEntries, reportError } = deps
  const { settings, readReadme } = api
  return {
    catalog: catalogEntries,
    health: createInstallationHealth(deps.health, reportError),
    retryPlugin: (pluginId: string) => retry.retryPlugin(pluginId),
    choosePath: async (pluginId: string, key: string): Promise<string> => {
      try {
        const answer = await settings.choosePluginPath(pluginId, key)
        if (answer.isChosen) {
          await rest.refresh()
        }
        return answer.problem
      } catch (caughtError) {
        return formatError(caughtError)
      }
    },
    files: createInstallationPluginFiles(settings, readReadme),
    plugins: rest.state.plugins,
    getSettingsViews,
    getIdentityViews: () => getIdentityViews(rest.state.plugins.value, getSettingsViews),
  }
}

function createInstallationConnections(
  connections: ConnectionStore,
  api: Pick<SettingsPageApi, 'listFiles' | 'listRowOptions'>,
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
      const answer = await api.listRowOptions(connectionId, field, options, operations)
      return answer.ok ? answer : { ok: false as const, error: answer.error.message }
    },
    countNotes: () => countNotes(api.listFiles),
  }
}

function createInstallationLicenses(licenses: LicenseStore) {
  return {
    list: licenses.licenses,
    load: () => licenses.load(),
    watch: (signal: AbortSignal) => licenses.watch(signal),
    activate: (licenseId: string, key: string) => licenses.activate(licenseId, key),
    deactivate: (licenseId: string) => licenses.deactivate(licenseId),
  }
}

function createInstallationIdentity(identity: IdentityStore) {
  return {
    identity: identity.identity,
    devices: identity.devices,
    load: () => identity.load(),
    generateWords: () => identity.generateWords(),
    create: (words: readonly string[]) => identity.create(words),
    restore: (words: readonly string[]) => identity.restore(words),
    delete: () => identity.delete(),
    createKit: (proof: KitProof) => identity.createKit(proof),
    checkKit: (sheets: readonly (readonly string[])[]) => identity.checkKit(sheets),
    restoreFromKit: (sheets: readonly (readonly string[])[]) => identity.restoreFromKit(sheets),
  }
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps): Installation {
  return {
    settings: createInstallationSettings(deps.settings),
    plugin: {
      ...createInstallationPlugin(deps.plugin),
      ...createInstallationDeclared(deps.plugin),
      ...createInstallationSlots(deps.plugin),
    },
    connections: createInstallationConnections(deps.connections, deps.api),
    licenses: createInstallationLicenses(deps.licenses),
    identity: createInstallationIdentity(deps.identity),
    printer: createPrinter(),
  }
}

/** The settings tab of a window: the page that edits numen.json, as a kind of tab. */
export function createSettingsPage({
  rest,
  connections,
  appearance,
  log,
  windowTabs,
  getPlatform,
  plugins,
  api,
}: SettingsPageDeps) {
  const getPluginName = (pluginId: string): string =>
    rest.state.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const reportError = (error: unknown) => log.getWriter('configured')(formatError(error), 'error')
  const licenses = createLicenseStore(api.license, (text, level) =>
    log.getWriter('licenses')(text, level),
  )
  const identity = createIdentityStore(api.identity, (text, level) =>
    log.getWriter('identity')(text, level),
  )
  const retry = createPluginRetry({
    askPluginRetry: api.settings.retryPlugin,
    words: { formatStopped: PLUGIN_FAILURE_WORDS.stopped, retryFailed: PLUGIN_FAILURE_WORDS.retryFailed },
    notify: log.getWriter('plugins'),
    refresh: () => rest.refresh(),
    getName: getPluginName,
  })

  const catalogEntries = useCatalogEntries(api.listCatalog, rest.state.plugins, reportError)

  const configured = useSettingsTabKind(
    windowTabs.handle,
    createSettingsTabConfig({
      settings: { appearance, rest, getPlatform, reportError },
      plugin: {
        api,
        rest,
        retry,
        getSettingsViews: plugins.getSettingsViews,
        ...(plugins.getSlotGroups ? { getSlotGroups: plugins.getSlotGroups } : {}),
        catalogEntries,
        reportError,
        health: plugins.health,
      },
      connections,
      licenses,
      identity,
      api,
    }),
    reportError,
  )
  return { configured, kinds: [configured.kind] }
}
