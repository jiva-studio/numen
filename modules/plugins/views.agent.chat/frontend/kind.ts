/**
 * The agent tab kind: a conversation with the agent, a tab of it per conversation.
 *
 * A question put to the window goes to the tab the person was last in, and a
 * window with none opens one. A command asked from a talk is asked over the
 * note the window relies on, since a talk is about no note of its own.
 */
import { Bot } from '@lucide/vue'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import AgentTab from './ui/AgentTab.vue'
import { createTabTitle } from './lib/title'
import { createAgentTabDeps, type AgentChatHost } from './model/tabDeps'
import type { AgentTabState } from './model/useAgentConversation'
import type { AgentTabDeps } from './types'
import { WORDS as words } from './words'

/** The word agent tabs are filed under. */
export const AGENT = 'agent'

/** What an agent tab is called: the first thing asked of it, shortened. */
const getAgentTitle = (state: AgentTabState): string =>
  createTabTitle(state.turns.value.find((turn) => turn.voice === 'prompt')?.text ?? '') ||
  words.agent

/** The agent kind as the window keeps it. Each tab holds a conversation made by `createState`. */
export function createAgentKind(
  host: AgentChatHost,
  createState: (deps: AgentTabDeps) => AgentTabState,
) {
  const kind: TabKindRegistration<AgentTabState> = {
    kind: AGENT,
    component: AgentTab,
    title: words.agent,
    icon: Bot,
    isTool: true,
    placement: { pane: 'aside', order: 0 },
    open: () => createState(createAgentTabDeps(host)),
    getTitle: getAgentTitle,
    onClose: (state) => {
      state.finish()
      return true
    },
    getTarget: () => host.workspace.getNoteInFocus(),
    receiveText: (state, text) => state.setQuestion(text),
  }
  return { kind }
}
