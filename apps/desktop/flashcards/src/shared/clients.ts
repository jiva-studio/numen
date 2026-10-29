/** The services the shell requests from the window that hosts it. */
import {
  agentService,
  notes,
  pluginService,
  vaultsService,
  windowService,
} from '@numen/plugin-sdk'

export { agentService, pluginService, vaultsService, windowService }

/** Where the links of a text land, in the vault a plugin names. */
export const noteService = notes
