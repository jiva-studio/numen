/**
 *
 * The window hands over what was written and reads what the agent does as it
 * does it. The port is `AgentPort`, and `@numen/plugin-sdk` says what the steps mean.
 */
import { createAgentPort } from '@numen/plugin-sdk'
import { agentService } from '@/shared/clients'

export const agentPort = createAgentPort(agentService)
