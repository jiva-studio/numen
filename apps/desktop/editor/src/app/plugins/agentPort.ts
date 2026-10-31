/**
 * The port through which the window sends what the user wrote and receives the
 * agent's steps as they happen. The port is `AgentPort`, and `@numen/plugin-sdk`
 * defines what the steps mean.
 */
import { createAgentPort } from '@numen/plugin-host'
import { agentService } from '@/shared/clients'

export const agentPort = createAgentPort(agentService)
