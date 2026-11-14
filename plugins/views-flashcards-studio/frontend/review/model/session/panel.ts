import { computed, shallowRef } from 'vue'
import type { ConnectionHost, ConversationContext } from '@numen/plugin-sdk'

import { WORDS as words } from '../../lib/session/agentWords'
import { getCardToolContext } from '../../api/session/agent'
import type { ReviewCard } from '../../lib/card/cards'

/** What the panel requires from the host window. */
export interface AgentPanelDeps {
  readonly connections: ConnectionHost
  readonly card: () => ReviewCard | null
  readonly vault: () => string
  readonly open: () => boolean
  readonly showPanel: (open: boolean) => void
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
   * Only the user's text is sent. The card goes to the tools as their context, and the vault the
   * tools work on is named, since the window has no vault of its own.
   */
  const startConversation = (targetCard: ReviewCard) => {
    const currentCard = current.value?.card
    if (
      currentCard?.address.cardId === targetCard.address.cardId &&
      currentCard.address.template === targetCard.address.template
    )
      return
    endConversation()
    current.value = {
      card: targetCard,
      context: {
        connections: deps.connections,
        toolContext: getCardToolContext(targetCard),
        vault: deps.vault(),
        placeholder: words.ask,
      },
    }
  }

  /** Opens the panel on the current card. */
  const openPanel = () => {
    const card = deps.card()
    if (!card) return
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
