/**
 * The agent of the window, which plugins query for what they show. Whether it is
 * reachable is queried once when the window opens; until the answer arrives, it
 * is assumed reachable.
 */
import { shallowRef } from 'vue'
import { createAgentPort, resolveTextLinks, type AgentHost } from '@numen/plugin-loader'
import { agentClient, getAgentUnreachable, linkClient } from '../api/agent'
import { WORDS } from '../words'

/** The agent of the window, and how its question about reachability is dropped. */
export function createWindowAgent(): {
  readonly agent: AgentHost
  readonly dispose: () => void
} {
  const port = createAgentPort(agentClient)
  const unreachable = shallowRef('')
  const aborting = new AbortController()
  getAgentUnreachable(aborting.signal)
    .then((reason) => {
      unreachable.value = reason
    })
    .catch(() => {
      if (!aborting.signal.aborted) unreachable.value = WORDS.agentUnreachable
    })

  const agent: AgentHost = {
    ask: (query) => port.ask(query),
    finish: (conversation) => port.finish(conversation),
    resolveLinks: (text, vault) => resolveTextLinks(linkClient, { text, vault }),
    getUnreachable: () => unreachable.value,
    getSetting: () => undefined,
    writeSetting: () => Promise.reject(new Error('this window keeps no agent settings')),
    getModelsAt: () => [],
  }
  return { agent, dispose: () => aborting.abort() }
}
