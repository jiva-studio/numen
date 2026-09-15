/**
 * The front doors of the application, and the window every question about a
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

/** This window itself, which is the one cards are run in and not the editor. */
export const itself = createClient(WindowService, transport)

/** The agent, which is the installation's and not a vault's. */
export const agentService = createClient(AgentService, transport)

/** The window every question about a window names. */
export const WINDOW = 'review'
