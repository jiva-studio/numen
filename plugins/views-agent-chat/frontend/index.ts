import {
  CONVERSATION,
  definePlugin,
  type HostServices,
  type PluginContext,
} from '@numen/plugin-sdk'
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

export function activate(plugin: PluginContext, host: HostServices): void {
  plugin.register.view({ capability: CONVERSATION, component: ConversationView })
  const { workspace } = host.ui
  if (!workspace) return
  const { connections } = host.work
  const { relations } = host.vault
  if (!connections || !relations) {
    throw new Error('an agent chat requires a window, its connections and the links of the vault')
  }
  const agents = createAgentList(connections, WORDS)
  const reloadAgents = () =>
    runInBackground(agents.reload(), (error) => host.ui.notices.show(String(error), 'error'))
  plugin.settings.onChange(AGENTS_SETTING, reloadAgents)
  // A host that tells of changes of the connections lists the agents now and again on each change; one
  // that does not lists them for each tab it opens.
  const hasChangeSignal = connections.onChange !== undefined
  connections.onChange?.(reloadAgents)
  if (hasChangeSignal) reloadAgents()
  const chats = createAgentKind(
    {
      workspace,
      relations,
      openSettings: createSettingsOpener(plugin, host),
    },
    (deps) => {
      if (!hasChangeSignal) reloadAgents()
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
  plugin.register.tabKind(chats.kind)

  plugin.register.command({
    id: 'chat.open',
    handler: async (note) => {
      await host.tabs.navigate?.(AGENT, note.path)
    },
  })
  /** A question about a note, written into the conversation the user was last in. */
  plugin.register.command({
    id: 'note.ask',
    handler: async (note) => {
      const id = await host.tabs.navigate?.(AGENT, note.path)
      if (id) chats.setDraft(id, `${note.path} — `)
    },
  })
}

export default definePlugin({
  activate,
})
