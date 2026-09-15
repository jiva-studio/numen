/**
 * Asking an agent about the vault the window is showing.
 *
 * The window hands over what was written and reads what the agent does as it
 * does it. The port is `AgentPort`, and `@numen/wire` says what the steps mean.
 */
import { agentPort } from '@numen/wire'
import { agentService } from '@/shared/clients'

export const core = agentPort(agentService)
