/**
 * The clients the window talks to the application through, made once from the transport the window
 * is given. A request carries no caller tag, so the core lists it as coming from the window, except
 * the plugin list, which is tagged as the plugins'.
 */
import {
  createCallerTransport,
  createConnectionCallClient,
  createConnectionCallRpc,
  createNoteRpc,
  createPluginRpc,
  createSettingsRpc,
  createThemeRpc,
  createVaultsRpc,
  createWindowRpc,
  type Transport,
} from '@numen/core-client'

export type { Transport }

export const createWindowClients = (transport: Transport) => ({
  notes: createNoteRpc(transport),
  connectionCalls: createConnectionCallClient(createConnectionCallRpc(transport)),
  pluginService: createPluginRpc(createCallerTransport('plugins', transport)),
  settingsService: createSettingsRpc(createCallerTransport('settings', transport)),
  theme: createThemeRpc(transport),
  vaultsService: createVaultsRpc(transport),
  windowService: createWindowRpc(transport),
})

export type WindowClients = ReturnType<typeof createWindowClients>
