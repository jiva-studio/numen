/**
 * The calls of the window to the core, made once from the transports the window is given and handed
 * to the parts that make them, grouped by what they reach: the vault, the window, the tabs, the
 * plugins and the configuration.
 */
import { createConnectionApi } from '@/entities/connection'
import { createFileApi } from '@/entities/file'
import { createLinkApi, createNoteApi, createRelations } from '@/entities/note'
import {
  createCatalogReader,
  createDeclaredCommandsApi,
  createHealthApi,
  createPluginHostApi,
  createProcessHost,
} from '@/entities/plugin'
import { createSettingsApi, createThemes } from '@/entities/settings'
import { createSavedTabsApi, createTabKindPluginFinder } from '@/entities/tab'
import { createVaults, createVaultStateApi } from '@/entities/vault'
import {
  createFocusApi,
  createQuitApi,
  createShownVaultApi,
  createTaskApi,
} from '@/entities/window'
import { createSearchCore } from '@/features/command-palette'
import { createTabLinksApi } from '@/features/tab-linking'
import { createTabMoveApi } from '@/features/tab-move'
import { createVaultListReader } from '@/features/window-display'
import {
  createPluginHostClients,
  createSettingsClients,
  createVaultClients,
  createWorkspaceClients,
  type WindowTransports,
} from '@/shared/api/clients'

export function createWindowApi(transports: WindowTransports) {
  const { transport } = transports
  const vaultClients = createVaultClients(transport)
  const workspaceClients = createWorkspaceClients(transports)
  const settingsClients = createSettingsClients(transport)
  const pluginClients = createPluginHostClients(transport)
  const links = createLinkApi(vaultClients.notes)
  const vaults = createVaults(vaultClients.vaultsService)
  const shownVault = createShownVaultApi(workspaceClients.windowService)
  return {
    vault: {
      note: createNoteApi(vaultClients.notes),
      links,
      relations: createRelations(vaultClients.notes, links.resolveLinks),
      file: createFileApi(vaultClients.files),
      vaults,
      vaultState: createVaultStateApi(vaultClients.vault),
      search: createSearchCore(vaultClients.search),
      listVaults: createVaultListReader({ vaults, getShownVault: shownVault.getShownVault }),
    },
    window: {
      focus: createFocusApi(workspaceClients.workspace),
      quit: createQuitApi(workspaceClients.windowService),
      shownVault,
      tasks: createTaskApi(workspaceClients.windowService),
    },
    tabs: {
      savedTabs: createSavedTabsApi(workspaceClients.workspace),
      tabLinks: createTabLinksApi(workspaceClients),
      tabMove: createTabMoveApi(workspaceClients.workspace),
    },
    plugins: {
      declaredCommands: createDeclaredCommandsApi(pluginClients.pluginCommands),
      pluginHost: createPluginHostApi({
        texts: vaultClients.texts,
        connectionCalls: pluginClients.connectionCalls,
      }),
      health: createHealthApi(pluginClients.pluginHealth),
      processes: createProcessHost(pluginClients.pluginService),
      listCatalog: createCatalogReader(pluginClients.pluginCatalog),
      findTabKindPlugin: createTabKindPluginFinder(pluginClients.pluginCatalog),
    },
    configuration: {
      connection: createConnectionApi(settingsClients.connections),
      settings: createSettingsApi(settingsClients.settingsService),
      themes: createThemes(settingsClients.theme),
    },
  }
}

/** Every call of the window to the core. */
export type WindowApi = ReturnType<typeof createWindowApi>
