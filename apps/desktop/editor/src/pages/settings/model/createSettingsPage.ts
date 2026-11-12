/**
 * The settings tab: the installation it edits and the kind of tab that shows it.
 */
import type { createWindowAppearance } from '@/features/settings-commands'
import {
  choosePluginPath,
  createPluginRetry,
  createSecretStore,
  SETTINGS_WORDS,
  deletePluginData,
  getPluginData,
  type SettingEdit,
  type SettingsStore,
} from '@/entities/settings'
import { PLUGIN_FAILURE_WORDS, readPluginReadme, useCatalogEntries } from '@/entities/plugin'
import { runInBackground } from '@numen/ui'
import { formatError, type Caught } from '@/shared/api/errors'
import { useSettingsTab } from '../kind'
import type { Installation, InstallationPlugin, InstallationSettings } from '../types'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'

export interface SettingsPageDeps {
  rest: SettingsStore
  appearance: ReturnType<typeof createWindowAppearance>
  log: MessageLog
  windowTabs: WindowTabs
  getSettingsViews: InstallationPlugin['getSettingsViews']
  getPlatform: InstallationSettings['getPlatform']
}

interface SettingsTabConfigDeps {
  appearance: ReturnType<typeof createWindowAppearance>
  rest: SettingsStore
  secrets: ReturnType<typeof createSecretStore>
  retry: ReturnType<typeof createPluginRetry>
  getSettingsViews: InstallationPlugin['getSettingsViews']
  getPlatform: InstallationSettings['getPlatform']
  catalogEntries: NonNullable<InstallationPlugin['catalog']>
  reportError: (error: Caught) => void
}

function createInstallationSettings(deps: SettingsTabConfigDeps) {
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
    getModels: (targetPath: readonly string[]) => rest.read.getModelsAt(targetPath),
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

function createInstallationPlugin(deps: SettingsTabConfigDeps) {
  const { rest, retry, getSettingsViews, catalogEntries } = deps
  return {
    catalog: catalogEntries,
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

function createInstallationSecrets({ rest, secrets }: SettingsTabConfigDeps) {
  return {
    sections: rest.state.secretSections,
    menus: rest.state.secretMenus,
    states: secrets.states,
    loadStates: () => secrets.loadStates(),
    credentialsPath: rest.state.credentialsPath,
    edit: secrets.edit,
    access: secrets.access,
  }
}

function createSettingsTabConfig(deps: SettingsTabConfigDeps): Installation {
  return {
    settings: createInstallationSettings(deps),
    plugin: createInstallationPlugin(deps),
    secrets: createInstallationSecrets(deps),
  }
}

/** The settings tab of a window: the page that edits numen.json, as a kind of tab. */
export function createSettingsPage({
  rest,
  appearance,
  log,
  windowTabs,
  getSettingsViews,
  getPlatform,
}: SettingsPageDeps) {
  const getPluginName = (pluginId: string): string =>
    rest.state.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const reportError = (error: Caught) => log.getWriter('configured')(formatError(error), 'error')
  const secrets = createSecretStore({
    words: SETTINGS_WORDS,
    notify: log.getWriter('keys'),
    refresh: () => rest.start(),
  })
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
      appearance,
      rest,
      secrets,
      retry,
      getSettingsViews,
      getPlatform,
      catalogEntries,
      reportError,
    }),
    reportError,
  )
  return { configured, kinds: [configured.kind] }
}
