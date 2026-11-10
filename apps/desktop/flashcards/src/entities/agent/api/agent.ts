/**
 * The agent service and the note service as the plugin host reads them. The host's agent port and
 * link resolver are typed by `@numen/plugin-loader`, so the clients pass through as they are.
 */
import type { AgentClient, LinkClient } from '@numen/plugin-loader'
import { agentService, notes } from '@/shared/api/clients'

/** The service that answers questions and finishes conversations. */
export const agentClient: AgentClient = agentService

/** Where the links of a text land, in the vault a plugin names. */
export const linkClient: LinkClient = notes

/** Why the agent cannot be reached, or an empty string where it can. */
export async function getAgentUnreachable(signal: AbortSignal): Promise<string> {
  return (await agentService.getAgentState({}, { signal })).unreachable
}
