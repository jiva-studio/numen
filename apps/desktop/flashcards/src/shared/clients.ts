/** The services the shell asks of the window it is the page of. */
import { createClient } from '@connectrpc/connect'
import { PluginService } from '@numen/protocol'
import { transport } from '@numen/wire'

/** The plugins of the window's target, whose frontends the shell loads. */
export const pluginService = createClient(PluginService, transport)
