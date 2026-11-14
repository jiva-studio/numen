/**
 * The clients the window talks to the application through, made once from the transport the window
 * is given. A request carries no caller tag, so the core lists it as coming from the window, except
 * the plugin list, which is tagged as the plugins'.
 */
import {
  createCallerTransport,
  createConnectionCallClient,
  createConnectionCallServiceClient,
  createNoteClient,
  createPluginClient,
  createThemeServiceClient,
  createVaultsClient,
  createWindowClient,
  type Transport,
} from '@numen/core-client'

export type { Transport }

export const createWindowClients = (transport: Transport) => ({
  notes: createNoteClient(transport),
  connectionCalls: createConnectionCallClient(createConnectionCallServiceClient(transport)),
  pluginService: createPluginClient(createCallerTransport('plugins', transport)),
  theme: createThemeServiceClient(transport),
  vaultsService: createVaultsClient(transport),
  windowService: createWindowClient(transport),
})

export type WindowClients = ReturnType<typeof createWindowClients>
