/** The agent conversation tab kind of a window. */
import { useConversation } from '@numen/ui'
import { agentKind, core as agent, useAgentConversation, WORDS as talk } from '@/pages/agent-chat'
import { CONVERSATION, minted } from '@/entities/tab'
import type { WindowKindsDeps } from './deps'

export interface AgentKindDeps extends Pick<WindowKindsDeps, 'core' | 'puts' | 'held' | 'window'> {
  about: () => { readonly path: string; readonly title: string }
}

export function createAgentKind({ core, puts, held, window, about }: AgentKindDeps) {
  return agentKind(
    held.handle,
    () =>
      useAgentConversation(useConversation(agent, talk, minted(CONVERSATION)), {
        opens: (path, ...runs) => void puts.opensAt(path, runs),
        beside: (path) => void puts.opens(path, '', 'beside'),
        resolve: (written) => core.resolve('', written),
        unreachable: () => window.unreachable.value,
      }),
    about,
  )
}
