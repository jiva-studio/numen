import {
  CONVERSATION,
  definePlugin,
  SETTINGS_OPEN_COMMAND,
  type Caught,
  type PluginContext,
} from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
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
  if (!workspace) return
  const { connections } = context.work
  const { relations } = context.vault
  if (!connections || !relations) {
    throw new Error('an agent chat requires a window, its connections and the links of the vault')
  }
  const chats = createAgentKind(
    {
      workspace,
      relations,
      openSettings: () => openPluginSettings(context),
    },
    (deps) =>
      useAgentConversation(
        useConversation({
          connections,
          words: WORDS,
          conversation: `conversation:${crypto.randomUUID()}`,
        }),
        deps,
      ),
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

/** Opens the settings page of this plugin through the shared command the window handles. */
function openPluginSettings(context: PluginContext): void {
  const { commands, notices } = context.ui
  if (!commands) return
  const values = { plugin: context.plugin.id }
  runInBackground(commands.run(SETTINGS_OPEN_COMMAND, undefined, undefined, values), (error: Caught) =>
    notices.show(error instanceof Error ? error.message : String(error), 'error'),
  )
}
