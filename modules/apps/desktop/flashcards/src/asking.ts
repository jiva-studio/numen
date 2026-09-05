/**
 * The panel a card is asked about in: whether it is showing, what has been said
 * in it, and what is being written.
 *
 * Everything the panel is, is here. What draws it reads this and decides
 * nothing: a card that may not be asked about is refused here, and the reason
 * is said here.
 *
 * A conversation belongs to one card. Answering the card ends it, so the card
 * in front of a person is never answered out of the one behind it.
 */
import { computed, ref, shallowRef } from 'vue'
import { conversation } from '@numen/ui'
import type { AgentPort, Conversation, Turn } from '@numen/ui'

import { WORDS as words } from './agent/words'
import type { CardFace } from './core'

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
  /** Whether the panel is what the window is showing. */
  readonly open: () => boolean
  /**
   * The panel asked for, or put away. The window can only be showing one thing,
   * so it is the window that holds which, and every panel moves that one thing.
   */
  readonly shows: (open: boolean) => void
  /** Where the window says what a person has to know. */
  readonly says: (said: string) => void
  /** When the words that have arrived are put on the screen. */
  readonly paint?: (draw: () => void) => void
}

export function asking(deps: AgentPanelDeps) {
  /** Whether the panel is what the window is showing, which the window holds. */
  const open = computed(() => deps.open())

  const written = ref('')

  /** The card the open conversation is about, and nothing while none is. */
  const about = shallowRef<CardFace | null>(null)

  /** The talk itself, made when a card is first asked about. */
  const talk = shallowRef<Conversation | null>(null)

  /** How many conversations this window has opened, which names the next. */
  let opened = 0

  const turns = computed<Turn[]>(() => talk.value?.turns.value ?? [])
  const working = computed(() => talk.value?.working.value ?? false)

  /** The talk about one card, made once and let go of with the card. */
  const talking = (card: CardFace) => {
    if (talk.value && about.value?.mark === card.mark && about.value?.face === card.face) return
    ends()
    about.value = card
    talk.value = conversation(deps.agent(card), words, `card-${opened++}`, deps.paint)
  }

  /**
   * The panel asked for, on whichever card is up. A window that can reach no
   * agent says so: a gesture that does nothing is a gesture a person repeats.
   */
  const opens = () => {
    const card = deps.card()
    if (!card) return
    const why = deps.unreachable()
    if (why) {
      deps.says(why)
      return
    }
    talking(card)
    deps.shows(true)
  }

  /** The panel put away, with what was said in it kept. */
  const shuts = () => {
    deps.shows(false)
  }

  const writing = (text: string) => {
    written.value = text
  }

  /**
   * A question sent about the card the panel stands on.
   *
   * What goes is what the person wrote and nothing else. Which card they are on
   * is the agent's to ask for: the deck, the mark and the face are named by a
   * vault that may have been synced from anywhere, and a name written into the
   * question is read as instruction.
   */
  const send = async (text: string) => {
    const card = about.value
    if (!card || !text) return
    written.value = ''
    await talk.value?.ask(text, card.deck)
  }

  const stop = () => talk.value?.stop()

  /** The conversation is over: the agent is told, and the panel holds nothing. */
  const ends = () => {
    talk.value?.finish()
    talk.value = null
    about.value = null
    written.value = ''
    deps.shows(false)
  }

  return {
    open,
    written,
    about,
    turns,
    working,
    opens,
    shuts,
    writing,
    send,
    stop,
    ends,
    unreachable: deps.unreachable,
  }
}

/** What one panel holds. */
export type AgentPanelState = ReturnType<typeof asking>
