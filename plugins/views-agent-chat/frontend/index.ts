import { CONVERSATION, definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import { useConversation } from './thread'
import AgentTab from './ui/AgentTab.vue'
import ConversationView from './ui/ConversationView.vue'
import { AGENT, createAgentKind } from './model/kind'
import { useAgentConversation } from './model/useAgentConversation'
import { createSettingsOpener } from './model/settings'
import { createAgentList, useAgentChoice } from './model/agents'
import { WORDS } from './words'

export { AgentTab }
export { AGENT, createAgentKind } from './model/kind'
export { ConversationView }
export { useAgentConversation } from './model/useAgentConversation'
export * from './types'
export { WORDS } from './words'

/** The key of the setting of the rows the turns of the chat go through. */
const AGENTS_SETTING = 'connections'

export function activate(context: PluginContext): void {
  context.register.view({ capability: CONVERSATION, component: ConversationView })
  const { workspace } = context.ui
  if (!workspace) return
  const { connections } = context.work
  const { relations } = context.vault
  if (!connections || !relations) {
    throw new Error('an agent chat requires a window, its connections and the links of the vault')
  }
  const agents = createAgentList(connections, WORDS.defaultModel)
  const reloadAgents = () =>
    runInBackground(agents.reload(), (error) => context.ui.notices.show(String(error), 'error'))
  context.settings.onChange(AGENTS_SETTING, reloadAgents)
  const chats = createAgentKind(
    {
      workspace,
      relations,
      openSettings: createSettingsOpener(context),
    },
    (deps) => {
      reloadAgents()
      const choice = useAgentChoice(connections, agents)
      const conversation = useConversation({
        connections,
        words: WORDS,
        conversation: `conversation:${crypto.randomUUID()}`,
        getPreferredRow: choice.getPreferredRow,
      })
      return useAgentConversation(conversation, deps, choice)
    },
    AgentTab,
  )
  context.register.tabKind(chats.kind)

  context.register.command({
    id: 'chat.open',
    handler: async (note) => {
      await context.tabs.navigate?.(AGENT, note.path)
    },
  })
  /** A question about a note, written into the conversation the user was last in. */
  context.register.command({
    id: 'note.ask',
    handler: async (note) => {
      const id = await context.tabs.navigate?.(AGENT, note.path)
      if (id) chats.setDraft(id, `${note.path} — `)
    },
  })
}

export default definePlugin({
  activate,
})
