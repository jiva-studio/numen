/**
 * The port through which the window sends what the user wrote and receives the
 * agent's steps as they happen. The port is `AgentPort`, and the plugin SDK
 * defines what the steps mean.
 */
import { createAgentPort } from '@numen/plugin-loader'
import { agentService } from '@/shared/api/clients'

export const agentPort = createAgentPort(agentService)
