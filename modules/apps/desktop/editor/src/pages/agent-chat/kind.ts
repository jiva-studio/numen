/**
 * Window registration for agent tabs.
 */
import type { TabKind, WindowHandle } from '@/entities/tab'
import { AGENT } from '@/entities/tab'
import AgentTab from './ui/AgentTab.vue'
import { WORDS as words } from './words'
import { firstLine } from './lib/title'
import type { AgentTabState } from './model/useAgentConversation'
import type { NoteRef } from './types'

/**
 * The agent tabs of a window, and the one the person was last in.
 *
 * A question about a note goes where the person was last talking, and a window
 * with no agent open opens one to carry it.
 *
 * A talk is about no note of its own, so a command asked from one is asked over
 * the note the plex the person was last in is standing on.
 */
export function agentKind(handle: WindowHandle, opens: () => AgentTabState, about: () => NoteRef) {
  const getTitle = (state: AgentTabState) =>
    firstLine(state.turns.value.find((turn) => turn.voice === 'asked')?.text ?? '') ||
    words.agent

  const kind: TabKind<AgentTabState, typeof AGENT> = {
    kind: AGENT,
    open: opens,
    getTitle,
    pane: AgentTab,
    onClose: (state) => {
      state.finish()
      return true
    },
    over: () => about(),
  }

  /** Something to ask, put in the agent the person was last in and put in front. */
  const askQuestion = async (text: string) => {
    const id = handle.last<AgentTabState>(AGENT)?.id ?? (await handle.openTab(AGENT))
    handle.getTabState<AgentTabState>(AGENT, id)?.setQuestion(text)
    handle.show(id)
  }

  return { kind, askQuestion }
}
