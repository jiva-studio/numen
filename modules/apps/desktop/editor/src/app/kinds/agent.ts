/** The agent conversation tab kind of a window. */
import { useConversation } from '@numen/ui'
import { agentKind, core as agent, useAgentConversation, WORDS as talk } from '@/pages/agent-chat'
import { CONVERSATION, generateId } from '@/entities/tab'
import type { WindowKindsDeps } from './deps'

export interface AgentKindDeps extends Pick<WindowKindsDeps, 'core' | 'tabOpeners' | 'held' | 'window'> {
  about: () => { readonly path: string; readonly title: string }
}

export function createAgentKind({ core, tabOpeners, held, window, about }: AgentKindDeps) {
  return agentKind(
    held.handle,
    () =>
      useAgentConversation(useConversation(agent, talk, generateId(CONVERSATION)), {
        openFileAt: (path, ...runs) => void tabOpeners.openFileAt(path, runs),
        openFileBeside: (path) => void tabOpeners.openFile(path, '', 'beside'),
        resolve: (written) => core.resolve('', written),
        unreachable: () => window.unreachable.value,
      }),
    about,
  )
}
