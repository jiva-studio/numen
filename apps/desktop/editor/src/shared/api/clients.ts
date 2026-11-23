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
  createContributionClient,
  createFileClient,
  createFolderDialogClient,
  buildIdentityClient,
  createIdentityServiceClient,
  createLogClient,
  createLogServiceClient,
  createLicenseClient,
  createLicenseServiceClient,
  createNoteClient,
  createPluginCatalog,
  createPluginDataClient,
  createPluginHealthClient,
  createPluginProcessClient,
  createProviderConnectionClient,
  createSearchClient,
  createSettingsClient,
  createTextClient,
  createTextVersionClient,
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
    folderDialog: createFolderDialogClient(transport),
  }
}

/** The clients of the workspace and of the window itself. */
export const createWorkspaceClients = ({ transport, keepalive }: WindowTransports) => ({
  workspace: createWorkspaceClient(transport),
  keepaliveWorkspace: createWorkspaceClient(keepalive),
  windowService: createWindowClient(transport),
  logService: createLogServiceClient(transport),
})

/** The clients of the settings, the connections, the licenses, the identity and the themes. */
export const createSettingsClients = (transport: Transport) => {
  const settings = createCallerTransport('settings', transport)
  return {
    settingsService: createSettingsClient(settings),
    connections: createConnectionClient(createProviderConnectionClient(settings)),
    licenses: createLicenseClient(createLicenseServiceClient(settings)),
    identity: buildIdentityClient(createIdentityServiceClient(settings)),
    pluginData: createPluginDataClient(settings),
    theme: createThemeServiceClient(transport),
  }
}

/** The clients the window runs and lists the plugins through. */
export const createPluginHostClients = (transport: Transport) => {
  const plugins = createCallerTransport('plugins', transport)
  const commands = createCallerTransport('commands', transport)
  return {
    pluginCommands: createCommandClient(commands),
    contributions: createContributionClient(commands),
    textVersions: createTextVersionClient(commands),
    pluginProcessService: createPluginProcessClient(plugins),
    pluginHealth: createPluginHealthClient(plugins),
    pluginCatalog: createPluginCatalog(createCatalogClient(transport)),
    connectionCalls: createConnectionCallClient(createConnectionCallServiceClient(transport)),
  }
}

/** Every client of the window. */
export type WindowClients = ReturnType<typeof createVaultClients> &
  ReturnType<typeof createWorkspaceClients> &
  ReturnType<typeof createSettingsClients> &
  ReturnType<typeof createPluginHostClients>

/** The application log used by the window. */
export type WindowLogClient = ReturnType<typeof createLogClient>

/** Creates the application log client over the window's service. */
export function createWindowLogClient(service: WindowClients['logService']): WindowLogClient {
  return createLogClient(service)
}
