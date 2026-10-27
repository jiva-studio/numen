/** The services the shell asks of the window it is the page of. */
import { createClient } from '@connectrpc/connect'
import {
  AgentService,
  NoteService,
  PluginService,
  VaultsService,
  WindowService,
} from '@numen/protocol'
import { transport } from '@numen/plugin-sdk'

/** The plugins of the window's target, whose frontends the shell loads. */
export const pluginService = createClient(PluginService, transport)

/** The vaults of the installation, which the window offers to choose from. */
export const vaultsService = createClient(VaultsService, transport)

/** The window itself, which reports the work behind it. */
export const windowService = createClient(WindowService, transport)

/** Where the links of a text land, in the vault a plugin names. */
export const noteService = createClient(NoteService, transport)

/** The agent of the installation, which the plugins ask about what they show. */
export const agentService = createClient(AgentService, transport)
