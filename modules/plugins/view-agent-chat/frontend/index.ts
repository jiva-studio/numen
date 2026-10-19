import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { useConversation } from '@numen/ui'
import AgentTab from './ui/AgentTab.vue'
import { createAgentKind } from './kind'
import { useAgentConversation } from './model/useAgentConversation'
import { WORDS } from './words'

export { AgentTab }
export { AGENT, createAgentKind } from './kind'
export { useAgentConversation } from './model/useAgentConversation'
export * from './types'
export { WORDS } from './words'

export function activate(context: PluginContext): void {
  const { workspace, agent } = context
  if (!workspace || !agent) {
    throw new Error('an agent chat needs the window it is drawn in and the agent it asks')
  }
  const chats = createAgentKind({ workspace, agent }, (deps) =>
    useAgentConversation(
      useConversation({
        agent,
        words: WORDS,
        conversation: `conversation:${crypto.randomUUID()}`,
      }),
      deps,
    ),
  )
  context.registerTabKind(chats.kind)
}

export default definePlugin({
  activate,
})
