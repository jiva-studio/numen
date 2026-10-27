/**
 * The agent of the window, which the plugins ask about what they show. Whether it can be reached
 * is asked once as the window opens; until the answer comes, it is taken to be reachable.
 */
import { shallowRef } from 'vue'
import {
  createAgentPort,
  resolveTextLinks,
  type AgentClient,
  type AgentHost,
  type LinkClient,
} from '@numen/plugin-sdk'
import { WORDS } from '@/shared/words'

/** As much of the agent service as the window asks of it. */
export interface WindowAgentClient extends AgentClient {
  getAgentState(
    request: Record<string, never>,
    options: { signal: AbortSignal },
  ): Promise<{ unreachable: string }>
}

/** The agent of the window, and how its question about reachability is dropped. */
export function createWindowAgent(
  client: WindowAgentClient,
  links: LinkClient,
): {
  readonly agent: AgentHost
  readonly dispose: () => void
} {
  const port = createAgentPort(client)
  const unreachable = shallowRef('')
  const aborting = new AbortController()
  client
    .getAgentState({}, { signal: aborting.signal })
    .then((state) => {
      unreachable.value = state.unreachable
    })
    .catch(() => {
      if (!aborting.signal.aborted) unreachable.value = WORDS.agentUnreachable
    })

  const agent: AgentHost = {
    ask: (query) => port.ask(query),
    finish: (conversation) => port.finish(conversation),
    resolveLinks: (text, vault) => resolveTextLinks(links, { text, vault }),
    getUnreachable: () => unreachable.value,
    getSetting: () => undefined,
    writeSetting: async () => {
      throw new Error('this window keeps no agent settings')
    },
    getModelsAt: () => [],
  }
  return { agent, dispose: () => aborting.abort() }
}
