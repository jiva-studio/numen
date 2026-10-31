/** The services the shell requests from the window that hosts it. */
import {
  agentService,
  notes,
  pluginService,
  theme,
  vaultsService,
  windowService,
} from '@numen/core-client'

export { agentService, pluginService, theme, vaultsService, windowService }

/** Where the links of a text land, in the vault a plugin names. */
export const noteService = notes
