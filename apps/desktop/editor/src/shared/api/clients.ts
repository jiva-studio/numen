/**
 * The clients the window talks to the application through, made once from the transports the
 * window is given.
 */
import {
  createCallerTransport,
  createCatalogClient,
  createCommandClient,
  createConnectionCallClient,
  createConnectionCallServiceClient,
  createConnectionClient,
  createFileClient,
  createNoteClient,
  createPluginCatalog,
  createPluginClient,
  createPluginHealthClient,
  createProviderConnectionClient,
  createSearchClient,
  createSettingsClient,
  createTextClient,
  createThemeServiceClient,
  createVaultClient,
  createVaultsClient,
  createWindowClient,
  createWorkspaceClient,
  type Transport,
} from '@numen/core-client'

export type { Transport }

/**
 * What the window sends its requests over. `keepalive` carries the requests that must complete
 * after the page unloads.
 */
export interface WindowTransports {
  readonly transport: Transport
  readonly keepalive: Transport
}

/**
 * Every client of the window. Each request carries the tag of the part of the window that makes
 * it, for the list of running plugins.
 */
export const createWindowClients = ({ transport, keepalive }: WindowTransports) => {
  const editor = createCallerTransport('editor', transport)
  const settings = createCallerTransport('settings', transport)
  const plugins = createCallerTransport('plugins', transport)
  return {
    vault: createVaultClient(editor),
    workspace: createWorkspaceClient(transport),
    keepaliveWorkspace: createWorkspaceClient(keepalive),
    files: createFileClient(editor),
    notes: createNoteClient(editor),
    texts: createTextClient(editor),
    search: createSearchClient(createCallerTransport('search', transport)),
    vaultsService: createVaultsClient(transport),
    settingsService: createSettingsClient(settings),
    connections: createConnectionClient(createProviderConnectionClient(settings)),
    windowService: createWindowClient(transport),
    theme: createThemeServiceClient(transport),
    pluginCommands: createCommandClient(createCallerTransport('commands', transport)),
    pluginService: createPluginClient(plugins),
    pluginHealth: createPluginHealthClient(plugins),
    pluginCatalog: createPluginCatalog(createCatalogClient(transport)),
    connectionCalls: createConnectionCallClient(createConnectionCallServiceClient(transport)),
  }
}

export type WindowClients = ReturnType<typeof createWindowClients>
