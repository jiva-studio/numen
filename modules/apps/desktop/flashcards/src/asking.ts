/**
 * The panel a card is asked about in: whether it is up, what has been said in
 * it, and what is being written.
 *
 * A conversation belongs to one card. Answering the card ends it, so the card
 * in front of a person is never answered out of the one behind it.
 */
import { computed, ref, shallowRef } from 'vue'
import { conversation } from '@numen/ui'
import type { AgentPort, Conversation, Turn } from '@numen/ui'

import { WORDS as words } from './agent/words'
import { standing } from './agent/core'
import type { Asked } from './core'

/** What the panel asks of the window it is drawn in. */
export interface Talking {
  /** What answers a question about a card. */
  readonly agent: AgentPort
  /** Why nothing can be asked here, empty while something can. */
  readonly unreachable: () => string
  /** The way in stands on every card whose answer is showing. */
  readonly everyCard: () => boolean
  /** When the words that have arrived are put on the screen. */
  readonly paint?: (draw: () => void) => void
}

export function asking(deps: Talking) {
  const up = ref(false)
  const written = ref('')

  /** The card the open conversation is about, and nothing while none is. */
  const about = shallowRef<Asked | null>(null)

  /** The talk itself, made when a card is first asked about. */
  const talk = shallowRef<Conversation | null>(null)

  /** How many conversations this window has opened, which names the next. */
  let opened = 0

  const turns = computed<Turn[]>(() => talk.value?.turns.value ?? [])
  const working = computed(() => talk.value?.working.value ?? false)

  /**
   * Whether the way into the panel stands on this card. It stands on the back
   * alone, and on a card the person could not recall unless the setting offers
   * it everywhere.
   */
  const offered = (shown: boolean, said: string): boolean => {
    if (!shown) return false
    return deps.everyCard() || said === 'again'
  }

  /** The talk about one card, made once and let go of with the card. */
  const talking = (card: Asked): Conversation => {
    if (talk.value && about.value?.card === card.card && about.value?.face === card.face) {
      return talk.value
    }
    ends()
    about.value = card
    talk.value = conversation(deps.agent, words, `card-${opened++}`, deps.paint)
    return talk.value
  }

  /** The panel comes in on a card whose answer is showing, and on no other. */
  const opens = (card: Asked, shown: boolean) => {
    if (!shown) return
    talking(card)
    up.value = true
  }

  const shuts = () => {
    up.value = false
  }

  const writing = (text: string) => {
    written.value = text
  }

  /**
   * A question sent about the card the panel stands on. The first of a
   * conversation carries which card that is; the rest are answered in the
   * thread it opened.
   */
  const send = async (text: string) => {
    const card = about.value
    if (!card || !text) return
    written.value = ''
    const first = turns.value.length === 0
    const asked = first ? `${standing(card)}\n\n${text}` : text
    await talk.value?.ask(asked, card.deck)
  }

  const stop = () => talk.value?.stop()

  /** The conversation is over: the agent is told, and the panel holds nothing. */
  const ends = () => {
    talk.value?.finish()
    talk.value = null
    about.value = null
    written.value = ''
    up.value = false
  }

  return {
    up,
    written,
    about,
    turns,
    working,
    offered,
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
export type Held = ReturnType<typeof asking>
