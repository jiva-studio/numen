/**
 * The services the review asks. Every question names the vault it is about: the window is over all
 * of them at once.
 */
import { createClient } from '@connectrpc/connect'
import { AgentService } from '@numen/protocol'
import { FlashcardsService } from '../../gen/numen/flashcards/v1/flashcards_pb.js'
import { transport } from '@numen/wire'

export const cards = createClient(FlashcardsService, transport)

/** The agent, which is the installation's and not a vault's. */
export const agentService = createClient(AgentService, transport)
