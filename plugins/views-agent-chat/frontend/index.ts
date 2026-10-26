import { Bot } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { useConversation } from '@numen/ui'
import AgentTab from './ui/AgentTab.vue'
import { AGENT, createAgentKind } from './kind'
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

  /** A question about a note, put in the conversation the person was last in. */
  const askAbout = (path: string): void => workspace.ask(`${path} — `)

  context.registerCommand({
    id: 'agent',
    title: WORDS.newAgent,
    shortcut: 'mod+shift+a',
    icon: Bot,
    handler: async () => {
      await context.openTab?.(AGENT)
    },
  })
  context.registerCommand({
    id: 'ask',
    title: WORDS.askAbout,
    scope: 'note',
    icon: Bot,
    handler: (note) => askAbout(note.path),
  })
  context.registerContextMenu({
    id: 'ask',
    text: WORDS.askAbout,
    group: 'agent',
    target: 'note',
    icon: Bot,
    handler: (target) => askAbout(target.path),
  })
}

export default definePlugin({
  activate,
})
