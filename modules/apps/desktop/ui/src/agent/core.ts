/**
 * Asking an agent about the vault the window is showing.
 *
 * The window hands over what was written and reads what the agent does as it
 * does it. The port is `AgentPort`, and `@numen/wire` says what the steps mean.
 */
import { createClient } from '@connectrpc/connect'
import { AgentService } from '@numen/protocol'
import { agentPort } from '@numen/wire'
import { transport } from '../transport'

export const core = agentPort(createClient(AgentService, transport))
