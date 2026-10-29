import { computed, shallowRef } from 'vue'
import type { AgentPort, ConversationContext } from '@numen/plugin-sdk'

import { WORDS as words } from '../lib/agentWords'
import type { ReviewCard } from '../../../entities/card'

/** What the panel requires from the host window. */
export interface AgentPanelDeps {
  /**
   * Returns the agent for a card. The card is attached to every question of
   * that conversation and is never part of the question text, so the agent
   * reads its name as a tool result.
   */
  readonly agent: (card: ReviewCard) => AgentPort
  /** The current card, and null between cards. */
  readonly card: () => ReviewCard | null
  /** The reason no question can be sent, empty while a question can be sent. */
  readonly unreachable: () => string
  readonly open: () => boolean
  /**
   * Opens or closes the panel. The window shows one panel at a time, so the
   * window holds which panel is open, and every panel changes that one state.
   */
  readonly showPanel: (open: boolean) => void
  /** Shows a message to the user. */
  readonly showNotice: (message: string) => void
}

/**
 * The panel for questions about a card. It holds the card the conversation is about
 * and the context the conversation view is mounted with; the conversation
 * itself belongs to the view and ends when the context is dropped.
 */
export function useAgentPanel(deps: AgentPanelDeps) {
  const open = computed(() => deps.open())

  /** The card in question and the context of its conversation, created when the first question about a card is sent. */
  const current = shallowRef<{ card: ReviewCard; context: ConversationContext } | null>(null)

  /** The card the open conversation is about, and nothing while none is. */
  const about = computed(() => current.value?.card ?? null)

  /** The context the conversation view is mounted with, null while no card is in question. */
  const context = computed(() => current.value?.context ?? null)

  /**
   * The conversation about one card, created once and ended with the card.
   *
   * Only the user's text is sent. The agent requests the card details itself:
   * the deck, the card id and the template name come from a vault that may have
   * been synced from anywhere, and a name written into the question is read as
   * an instruction.
   */
  const startConversation = (targetCard: ReviewCard) => {
    const currentCard = current.value?.card
    if (currentCard?.cardId === targetCard.cardId && currentCard.template === targetCard.template)
      return
    endConversation()
    current.value = {
      card: targetCard,
      context: {
        agent: deps.agent(targetCard),
        focus: targetCard.deck,
        placeholder: words.ask,
        getUnreachable: deps.unreachable,
      },
    }
  }

  /**
   * Opens the panel on the current card. When no agent is reachable the panel
   * shows a notice, because a control that does nothing is pressed repeatedly.
   */
  const openPanel = () => {
    const card = deps.card()
    if (!card) return
    const why = deps.unreachable()
    if (why) return deps.showNotice(why)
    startConversation(card)
    deps.showPanel(true)
  }

  /** The panel put away, with the transcript in it kept. */
  const closePanel = () => deps.showPanel(false)

  /** The conversation is over: the view goes, and with it the conversation. */
  const endConversation = () => {
    current.value = null
    deps.showPanel(false)
  }

  return { open, about, context, openPanel, closePanel, endConversation }
}

/** What one panel holds. */
export type AgentPanelState = ReturnType<typeof useAgentPanel>
