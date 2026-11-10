import { CONVERSATION, definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { useConversation } from './thread'
import AgentTab from './ui/AgentTab.vue'
import ConversationView from './ui/ConversationView.vue'
import { AGENT, createAgentKind } from './model/kind'
import { useAgentConversation } from './model/useAgentConversation'
import { WORDS } from './words'

export { AgentTab }
export { AGENT, createAgentKind } from './model/kind'
export { ConversationView }
export { useAgentConversation } from './model/useAgentConversation'
export * from './types'
export { WORDS } from './words'

export function activate(context: PluginContext): void {
  context.register.view({ capability: CONVERSATION, component: ConversationView })
  const { workspace } = context.ui
  const { agent } = context.work
  if (!workspace) return
  if (!agent) {
    throw new Error('an agent chat requires a window and an agent')
  }
  const chats = createAgentKind(
    {
      workspace,
      agent,
      ...(context.ui.commands ? { commands: context.ui.commands } : {}),
      getReasoning: () => context.settings.get<string>('reasoning') ?? '',
    },
    (deps) =>
      useAgentConversation(
        useConversation({
          agent,
          words: WORDS,
          conversation: `conversation:${crypto.randomUUID()}`,
        }),
        deps,
      ),
    AgentTab,
  )
  context.register.tabKind(chats.kind)

  /** A question about a note, posted to the conversation the user was last in. */
  const askAbout = (path: string): void => workspace.ask(`${path} — `)

  context.register.command({
    id: 'chat.open',
    handler: async (note) => {
      await context.tabs.navigate?.(AGENT, note.path)
    },
  })
  context.register.command({
    id: 'note.ask',
    handler: (note) => askAbout(note.path),
  })
}

export default definePlugin({
  activate,
})
