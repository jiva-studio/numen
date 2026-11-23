/**
 * The clients the window talks to the application through, made once from the transports the
 * window is given. Each request carries the tag of the part of the window that makes it, for the
 * list of running plugins.
 */
import {
  createCallerTransport,
  createCatalogRpc,
  createCommandRpc,
  createConnectionCallClient,
  createConnectionCallRpc,
  createConnectionClient,
  createContributionRpc,
  createFileRpc,
  createFolderDialogRpc,
  createLogClient,
  createLogRpc,
  createLicenseClient,
  createLicenseRpc,
  createNoteRpc,
  createPluginCatalog,
  createPluginDataRpc,
  createPluginHealthRpc,
  createPluginProcessRpc,
  createProviderConnectionRpc,
  createSearchRpc,
  createSettingsRpc,
  createTextRpc,
  createTextVersionRpc,
  createThemeRpc,
  createVaultRpc,
  createVaultsRpc,
  createWindowRpc,
  createWorkspaceRpc,
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
    vault: createVaultRpc(editor),
    files: createFileRpc(editor),
    notes: createNoteRpc(editor),
    texts: createTextRpc(editor),
    search: createSearchRpc(createCallerTransport('search', transport)),
    vaultsService: createVaultsRpc(transport),
    folderDialog: createFolderDialogRpc(transport),
  }
}

/** The clients of the workspace and of the window itself. */
export const createWorkspaceClients = ({ transport, keepalive }: WindowTransports) => ({
  workspace: createWorkspaceRpc(transport),
  keepaliveWorkspace: createWorkspaceRpc(keepalive),
  windowService: createWindowRpc(transport),
  logService: createLogRpc(transport),
})

/** The clients of the settings, the connections, the licenses and the themes. */
export const createSettingsClients = (transport: Transport) => {
  const settings = createCallerTransport('settings', transport)
  return {
    settingsService: createSettingsRpc(settings),
    connections: createConnectionClient(createProviderConnectionRpc(settings)),
    licenses: createLicenseClient(createLicenseRpc(settings)),
    pluginData: createPluginDataRpc(settings),
    theme: createThemeRpc(transport),
  }
}

/** The clients the window runs and lists the plugins through. */
export const createPluginHostClients = (transport: Transport) => {
  const plugins = createCallerTransport('plugins', transport)
  const commands = createCallerTransport('commands', transport)
  return {
    pluginCommands: createCommandRpc(commands),
    contributions: createContributionRpc(commands),
    textVersions: createTextVersionRpc(commands),
    pluginProcessService: createPluginProcessRpc(plugins),
    pluginHealth: createPluginHealthRpc(plugins),
    pluginCatalog: createPluginCatalog(createCatalogRpc(transport)),
    connectionCalls: createConnectionCallClient(createConnectionCallRpc(transport)),
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
