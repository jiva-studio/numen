/**
 * The services the review asks. Every question names the vault it is about: the window is over all
 * of them at once.
 */
import { createClient } from '@connectrpc/connect'
import { AgentService, FlashcardsService, WindowService } from '@numen/protocol'
import { transport } from '@numen/wire'
import { FLASHCARDS_TARGET } from '../target'

export const cards = createClient(FlashcardsService, transport)

/** The window the review is drawn in, which reports the work behind it. */
export const windowService = createClient(WindowService, transport)

/** The agent, which is the installation's and not a vault's. */
export const agentService = createClient(AgentService, transport)

/** The name of the window the review is drawn in, which is the target it is activated in. */
export const WINDOW = FLASHCARDS_TARGET
