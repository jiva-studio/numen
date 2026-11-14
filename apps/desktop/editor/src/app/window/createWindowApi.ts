/**
 * The calls of the window to the core, made once from the transports the window is given and handed
 * to the parts that make them.
 */
import { createConnectionApi } from '@/entities/connection'
import { createFileApi } from '@/entities/file'
import { createNoteApi, createRelations } from '@/entities/note'
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
import { createWindowClients, type WindowTransports } from '@/shared/api/clients'

export function createWindowApi(transports: WindowTransports) {
  const clients = createWindowClients(transports)
  const note = createNoteApi(clients.notes)
  return {
    connection: createConnectionApi(clients.connections),
    file: createFileApi(clients.files),
    note,
    relations: createRelations(clients.notes, note.resolveLinks),
    declaredCommands: createDeclaredCommandsApi(clients.pluginCommands),
    pluginHost: createPluginHostApi(clients),
    health: createHealthApi(clients.pluginHealth),
    processes: createProcessHost(clients.pluginService),
    listCatalog: createCatalogReader(clients.pluginCatalog),
    settings: createSettingsApi(clients.settingsService),
    themes: createThemes(clients.theme),
    savedTabs: createSavedTabsApi(clients.workspace),
    findTabKindPlugin: createTabKindPluginFinder(clients.pluginCatalog),
    vaults: createVaults(clients.vaultsService),
    vaultState: createVaultStateApi(clients.vault),
    focus: createFocusApi(clients.workspace),
    quit: createQuitApi(clients.windowService),
    shownVault: createShownVaultApi(clients.windowService),
    tasks: createTaskApi(clients.windowService),
    search: createSearchCore(clients.search),
    tabLinks: createTabLinksApi(clients),
    tabMove: createTabMoveApi(clients.workspace),
  }
}

/** Every call of the window to the core. */
export type WindowApi = ReturnType<typeof createWindowApi>
