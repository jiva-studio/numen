/**
 * The clients the window talks to the application through, made once from the transports the
 * window is given. Each request carries the tag of the part of the window that makes it, for the
 * list of running plugins.
 */
import {
  createCallerTransport,
  createCatalogClient,
  createCommandClient,
  createConnectionCallClient,
  createConnectionCallServiceClient,
  createConnectionClient,
  createFileClient,
  createLogServiceClient,
  createLicenseClient,
  createLicenseServiceClient,
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

/**
 * What the window sends its requests over. `keepalive` carries the requests that must complete
 * after the page unloads.
 */
export interface WindowTransports {
  readonly transport: Transport
  readonly keepalive: Transport
}

/** The clients of the vault the window shows and of the vaults it can open. */
export const createVaultClients = (transport: Transport) => {
  const editor = createCallerTransport('editor', transport)
  return {
    vault: createVaultClient(editor),
    files: createFileClient(editor),
    notes: createNoteClient(editor),
    texts: createTextClient(editor),
    search: createSearchClient(createCallerTransport('search', transport)),
    vaultsService: createVaultsClient(transport),
  }
}

/** The clients of the workspace and of the window itself. */
export const createWorkspaceClients = ({ transport, keepalive }: WindowTransports) => ({
  workspace: createWorkspaceClient(transport),
  keepaliveWorkspace: createWorkspaceClient(keepalive),
  windowService: createWindowClient(transport),
})

/** The clients of the settings, the connections, the licenses and the themes. */
export const createSettingsClients = (transport: Transport) => {
  const settings = createCallerTransport('settings', transport)
  return {
    settingsService: createSettingsClient(settings),
    connections: createConnectionClient(createProviderConnectionClient(settings)),
    licenses: createLicenseClient(createLicenseServiceClient(settings)),
    theme: createThemeServiceClient(transport),
  }
}

/** The clients the window runs and lists the plugins through. */
export const createPluginHostClients = (transport: Transport) => {
  const plugins = createCallerTransport('plugins', transport)
  return {
    pluginCommands: createCommandClient(createCallerTransport('commands', transport)),
    pluginService: createPluginClient(plugins),
    pluginHealth: createPluginHealthClient(plugins),
    pluginCatalog: createPluginCatalog(createCatalogClient(transport)),
    connectionCalls: createConnectionCallClient(createConnectionCallServiceClient(transport)),
    logService: createLogServiceClient(plugins),
  }
}

/** Every client of the window. */
export type WindowClients = ReturnType<typeof createVaultClients> &
  ReturnType<typeof createWorkspaceClients> &
  ReturnType<typeof createSettingsClients> &
  ReturnType<typeof createPluginHostClients>
