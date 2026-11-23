/**
 * The calls of the window to the core, made once from the transports the window is given and handed
 * to the parts that make them, grouped by what they reach: the vault, the window, the tabs, the
 * plugins and the configuration.
 */
import { createFrameRuntimeLoader } from '@numen/plugin-loader'
import { createConnectionApi } from '@/entities/connection'
import { createFileApi } from '@/entities/file'
import { createLicenseApi } from '@/entities/license'
import { createLinkApi, createNoteApi, createRelations } from '@/entities/note'
import {
  createCatalogReader,
  createReadmeReader,
  createDeclaredCommandsApi,
  createHealthApi,
  createLogHost,
  createPluginHostApi,
  createProcessHost,
} from '@/entities/plugin'
import { createSettingsApi, createThemes } from '@/entities/settings'
import { createSavedTabsApi, createTabKindPluginFinder } from '@/entities/tab'
import { createVaultListReader, createVaults, createVaultStateApi } from '@/entities/vault'
import {
  createFocusApi,
  createQuitApi,
  createShownVaultApi,
  createTaskApi,
} from '@/entities/window'
import { createSearchCore } from '@/features/command-palette'
import { createTabLinksApi } from '@/features/tab-linking'
import { createTabMoveApi } from '@/features/tab-move'
import {
  createPluginHostClients,
  createSettingsClients,
  createVaultClients,
  createWorkspaceClients,
  createWindowLogClient,
  type WindowTransports,
} from '@/shared/api/clients'

export function createWindowApi(transports: WindowTransports) {
  const { transport } = transports
  const vaultClients = createVaultClients(transport)
  const workspaceClients = createWorkspaceClients(transports)
  const settingsClients = createSettingsClients(transport)
  const pluginClients = createPluginHostClients(transport)
  const linkApi = createLinkApi(vaultClients.notes)
  const vaults = createVaults(vaultClients.vaultsService, vaultClients.folderDialog)
  const windowVaultApi = createShownVaultApi(workspaceClients.windowService)
  return {
    vault: {
      note: createNoteApi(vaultClients.notes),
      links: linkApi,
      relations: createRelations(vaultClients.notes, linkApi.resolveLinks),
      file: createFileApi(vaultClients.files),
      vaults,
      vaultState: createVaultStateApi(vaultClients.vault),
      search: createSearchCore(vaultClients.search),
      listVaults: createVaultListReader({ vaults, getShownVault: windowVaultApi.getShownVault }),
    },
    window: {
      logs: createWindowLogClient(workspaceClients.logService),
      focus: createFocusApi(workspaceClients.workspace),
      quit: createQuitApi(workspaceClients.windowService),
      shownVault: windowVaultApi,
      tasks: createTaskApi(workspaceClients.windowService),
      frameRuntime: createFrameRuntimeLoader(),
    },
    tabs: {
      savedTabs: createSavedTabsApi(workspaceClients.workspace),
      tabLinks: createTabLinksApi(workspaceClients),
      tabMove: createTabMoveApi(workspaceClients.workspace),
    },
    plugins: {
      declaredCommands: createDeclaredCommandsApi(
        pluginClients.pluginCommands,
        pluginClients.contributions,
        pluginClients.textVersions,
      ),
      pluginHost: createPluginHostApi({
        texts: vaultClients.texts,
        connectionCalls: pluginClients.connectionCalls,
      }),
      health: createHealthApi(pluginClients.pluginHealth),
      processes: createProcessHost(pluginClients.pluginProcessService),
      logs: createLogHost(workspaceClients.logService),
      listCatalog: createCatalogReader(pluginClients.pluginCatalog),
      readReadme: createReadmeReader(pluginClients.pluginCatalog),
      findTabKindPlugin: createTabKindPluginFinder(pluginClients.pluginCatalog),
    },
    configuration: {
      connection: createConnectionApi(settingsClients.connections),
      license: createLicenseApi(settingsClients.licenses),
      settings: createSettingsApi(
        settingsClients.settingsService,
        pluginClients.pluginProcessService,
        settingsClients.pluginData,
      ),
      themes: createThemes(settingsClients.theme),
    },
  }
}

/** Every call of the window to the core. */
export type WindowApi = ReturnType<typeof createWindowApi>
