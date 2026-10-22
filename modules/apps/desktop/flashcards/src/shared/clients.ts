/**
 * window names.
 */
import { createClient } from '@connectrpc/connect'
import { AgentService, FlashcardsService, WindowService } from '@numen/protocol'
import { transport } from '@numen/wire'

/**
 * What this window asks, presets among it. Every question names the vault it is
 * about: this window is over all of them at once.
 */
export const cards = createClient(FlashcardsService, transport)

/** The service of this window, the one cards are run in. */
export const windowService = createClient(WindowService, transport)

/** The agent, which is the installation's and not a vault's. */
export const agentService = createClient(AgentService, transport)

/** The window every question about a window names. */
export const WINDOW = 'review'
