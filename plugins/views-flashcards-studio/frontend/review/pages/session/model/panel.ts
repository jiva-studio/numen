import { computed, shallowRef } from 'vue'
import type { AgentPort, ConversationContext } from '@numen/plugin-sdk'

import { WORDS as words } from '../lib/agentWords'
import type { ReviewCard } from '../../../entities/card'

/** What the panel asks of the window it is drawn in. */
export interface AgentPanelDeps {
  /**
   * What answers a question about a card, asked for the card it is about. The
   * card travels beside every question of that conversation and never inside
   * one, so the agent reads its name as a tool's answer.
   */
  readonly agent: (card: ReviewCard) => AgentPort
  /** The card in front of the person, and nothing between cards. */
  readonly card: () => ReviewCard | null
  /** Why nothing can be asked here, empty while something can. */
  readonly unreachable: () => string
  readonly open: () => boolean
  /**
   * The panel asked for, or put away. The window can only be showing one thing,
   * so it is the window that holds which, and every panel moves that one thing.
   */
  readonly showPanel: (open: boolean) => void
  /** Where the window says what a person has to know. */
  readonly showNotice: (message: string) => void
}

/**
 * The panel a card is asked about in. It holds which card the conversation is about and the
 * context the conversation view is mounted with; the conversation itself is the view's, and ends
 * when the context goes.
 */
export function useAgentPanel(deps: AgentPanelDeps) {
  const open = computed(() => deps.open())

  /** The card asked about, and the context of its conversation, made when a card is first asked about. */
  const current = shallowRef<{ card: ReviewCard; context: ConversationContext } | null>(null)

  /** The card the open conversation is about, and nothing while none is. */
  const about = computed(() => current.value?.card ?? null)

  /** What the conversation view is mounted with, and nothing while no card is asked about. */
  const context = computed(() => current.value?.context ?? null)

  /**
   * The conversation about one card, made once and ended with the card.
   *
   * What is sent is what the person wrote and nothing else. Which card they are on is the agent's
   * to ask for: the deck, the card id and the template are named by a vault that may have been synced from
   * anywhere, and a name written into the question is read as instruction.
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
   * The panel asked for, on whichever card is up. A window that can reach no
   * agent says so: a gesture that does nothing is a gesture a person repeats.
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
