/**
 * The agent tab kind: a conversation with the agent, a tab of it per conversation.
 *
 * A question sent to the window goes to the most recently focused tab, and the
 * window opens a tab when there is none. A command issued from a conversation
 * applies to the note the window currently tracks, since a conversation has no
 * note of its own.
 */
import { Bot } from '@lucide/vue'
import type { Component } from 'vue'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import { createTabTitle } from '../lib/title'
import { createSavedConversation, parseSavedConversation } from '../lib/saved'
import { createAgentTabDeps, type AgentChatHost } from './tabDeps'
import type { AgentTabState } from './useAgentConversation'
import type { AgentTabDeps } from '../types'
import { WORDS as words } from '../words'

/** The word agent tabs are filed under. */
export const AGENT = 'agent'

const getAgentTitle = (state: AgentTabState): string =>
  createTabTitle(state.addresses.turns.value.find((turn) => turn.voice === 'prompt')?.text ?? '') ||
  words.agent

/**
 * The agent kind as the window keeps it. Each tab holds a conversation made by `createState` and is
 * drawn by `component`; `setDraft` writes the question of the tab of an id.
 */
export function createAgentKind(
  host: AgentChatHost,
  createState: (deps: AgentTabDeps) => AgentTabState,
  component: Component,
) {
  const states = new Map<string, AgentTabState>()
  const kind: TabKindRegistration<AgentTabState> = {
    kind: AGENT,
    component,
    title: words.agent,
    icon: Bot,
    isTool: true,
    placement: { pane: 'aside', order: 0 },
    open: (_targetPath, id, saved) => {
      const state = createState(createAgentTabDeps(host))
      state.draft.restore(parseSavedConversation(saved))
      states.set(id, state)
      return state
    },
    getState: (state) =>
      createSavedConversation(
        state.draft.userQuestion.value,
        state.addresses.turns.value,
        state.draft.getSavedModel(),
      ),
    getTitle: getAgentTitle,
    onClose: (state, id) => {
      state.conversation.stop()
      states.delete(id)
      return true
    },
    onDestroy: (state, id) => {
      state.conversation.stop()
      states.delete(id)
    },
    getTarget: () => host.workspace.getNoteInFocus(),
    receiveText: (state, text) => state.draft.setQuestion(text),
  }
  const setDraft = (id: string, text: string): void => states.get(id)?.draft.setQuestion(text)
  return { kind, setDraft }
}
