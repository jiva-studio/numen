import { computed, ref, shallowRef } from 'vue'
import { useConversation } from '@numen/ui'
import type { AgentPort, Conversation, Turn } from '@numen/ui'

import { WORDS as words } from '../lib/agentWords'
import type { CardFace } from '@/entities/card'

/** What the panel asks of the window it is drawn in. */
export interface AgentPanelDeps {
  /**
   * What answers a question about a card, asked for the card it is about. The
   * card travels beside every question of that conversation and never inside
   * one, so the agent reads its name as a tool's answer.
   */
  readonly agent: (card: CardFace) => AgentPort
  /** The card in front of the person, and nothing between cards. */
  readonly card: () => CardFace | null
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
  /** When the words that have arrived are put on the screen. */
  readonly paint?: (draw: () => void) => void
}

function createCardConversation(
  agent: AgentPort,
  conversationId: string,
  paint?: (draw: () => void) => void,
): Conversation {
  return useConversation({ agent, words, conversation: conversationId, paint })
}

export function useAgentPanel(deps: AgentPanelDeps) {
  const open = computed(() => deps.open())

  const content = ref('')

  /** The open conversation with the card it is about, made when a card is first asked about. */
  const card = shallowRef<{ card: CardFace; conversation: Conversation } | null>(null)

  /** The card the open conversation is about, and nothing while none is. */
  const about = computed(() => card.value?.card ?? null)

  /** How many conversations this window has opened, which names the next. */
  let conversationCounter = 0

  const turns = computed<Turn[]>(() => card.value?.conversation.turns.value ?? [])
  const isWorking = computed(() => card.value?.conversation.isWorking.value ?? false)

  /** The conversation about one card, made once and release with the card. */
  const startTalk = (targetCard: CardFace) => {
    if (card.value?.card.mark === targetCard.mark && card.value.card.face === targetCard.face) return
    endConversation()
    const conversation = createCardConversation(deps.agent(targetCard), `card-${conversationCounter++}`, deps.paint)
    card.value = { card: targetCard, conversation }
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
    startTalk(card)
    deps.showPanel(true)
  }

  /** The panel put away, with the transcript in it kept. */
  const closePanel = () => deps.showPanel(false)

  const setWritten = (text: string) => {
    content.value = text
  }

  /**
   * A question sent about the card the panel relies on.
   *
   * What goes is what the person wrote and nothing else. Which card they are on
   * is the agent's to ask for: the deck, the mark and the face are named by a
   * vault that may have been synced from anywhere, and a name written into the
   * question is read as instruction.
   */
  const send = async (text: string) => {
    const current = card.value
    if (!current || !text) return
    content.value = ''
    await current.conversation.ask(text, current.card.deck)
  }

  const stop = () => card.value?.conversation.stop()

  /** The conversation is over: the agent is told, and the panel holds nothing. */
  const endConversation = () => {
    card.value?.conversation.finish()
    card.value = null
    content.value = ''
    deps.showPanel(false)
  }

  return {
    open,
    written: content,
    about,
    turns,
    isWorking,
    openPanel,
    closePanel,
    setWritten,
    send,
    stop,
    endConversation,
    unreachable: deps.unreachable,
  }
}

/** What one panel holds. */
export type AgentPanelState = ReturnType<typeof useAgentPanel>
