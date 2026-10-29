/**
 * The agent tab kind: a conversation with the agent, a tab of it per conversation.
 *
 * A question sent to the window goes to the most recently focused tab, and the
 * window opens a tab when there is none. A command issued from a conversation
 * applies to the note the window currently tracks, since a conversation has no
 * note of its own.
 */
import { Bot } from '@lucide/vue'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import AgentTab from './ui/AgentTab.vue'
import { createTabTitle } from './lib/title'
import { createSavedConversation, parseSavedConversation } from './lib/saved'
import { createAgentTabDeps, type AgentChatHost } from './model/tabDeps'
import type { AgentTabState } from './model/useAgentConversation'
import type { AgentTabDeps } from './types'
import { WORDS as words } from './words'

/** The word agent tabs are filed under. */
export const AGENT = 'agent'

/** The title of an agent tab: the first prompt, shortened. */
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
    open: (_targetPath, _id, saved) => {
      const state = createState(createAgentTabDeps(host))
      state.restore(parseSavedConversation(saved))
      return state
    },
    getState: (state) => createSavedConversation(state.userQuestion.value, state.turns.value),
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
