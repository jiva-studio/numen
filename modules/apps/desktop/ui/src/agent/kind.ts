/**
 * What one agent tab holds: a talk of its own, and what pressing anything in
 * it comes to.
 *
 * A question carries the note the person is looking at. A line about work and
 * a link inside an answer both name a place in a source, and both open it.
 */
import { ref } from 'vue'
import type { Turn } from '@numen/ui'
import type { Conversation } from '../conversation'
import { same, spotOf, spotsIn } from '../places'
import type { Run } from '../reading'
import type { Host, Kind } from '../windowing'
import { WORDS as words } from '../words'
import { AGENT, shortened } from '../workspace'
import AgentTab from './AgentTab.vue'

/** What an agent tab asks of the window it is drawn in. */
export interface Talking {
  /** The note the person is looking at, which is what a question is about. */
  looking(): string
  /** A source opened at stretches of its own text, the first of them in front. */
  opens(path: string, ...runs: readonly Run[]): void
  /** Why the agent cannot be reached, which the tab says where its answers stand. */
  unreachable(): string
}

/** What one agent tab holds. */
export type Held = ReturnType<typeof talking>

export function talking(talk: Conversation, deps: Talking) {
  /** The question being written, until it is sent. */
  const asked = ref('')

  const writing = (text: string) => {
    asked.value = text
  }

  /** A question sent, about the note the person is looking at. */
  const send = (text: string) => {
    asked.value = ''
    void talk.ask(text, deps.looking())
  }

  /** A line about work pressed: the place that call was on is put in front. */
  const opensTurn = (turn: Turn) => {
    const place = talk.place(turn.id)
    if (place) deps.opens(place.path, { start: place.start, length: place.length })
  }

  /**
   * A link inside an answer pressed. One naming a place in the vault opens it,
   * and the other places that answer names in the same document are lit with
   * it. Any other link is left to whatever would follow it.
   */
  const followed = (turn: Turn, href: string, press: MouseEvent) => {
    const here = spotOf(href)
    if (!here) return
    press.preventDefault()
    const named = spotsIn(turn.text).filter((spot) => spot.path === here.path && !same(spot, here))
    deps.opens(here.path, ...[here, ...named].map(({ start, length }) => ({ start, length })))
  }

  return { ...talk, asked, writing, send, opensTurn, followed, unreachable: deps.unreachable }
}

/**
 * The agent tabs of a window, and the one the person was last in.
 *
 * A question about a note goes where the person was last talking, and a window
 * with no agent open opens one to carry it.
 */
export function agentKind(host: Host, opens: () => Held) {
  const talks = new Map<string, Held>()
  /** The agent tab the person was last in, while the window still holds it. */
  let last = ''

  const kind: Kind<Held> = {
    kind: AGENT,
    opens: (_at, id) => {
      const held = opens()
      talks.set(id, held)
      return held
    },
    called: (held) =>
      shortened(held.turns.value.find((turn) => turn.voice === 'asked')?.text ?? '') || words.agent,
    draws: AgentTab,
    shown: (_held, id) => {
      last = id
    },
    shuts: (held, id) => {
      held.finish()
      talks.delete(id)
      if (last === id) last = ''
      return true
    },
    offers: words.newAgent,
  }

  /** Something to ask, put in the agent the person was last in and put in front. */
  const asks = async (text: string) => {
    const id = talks.has(last) ? last : await host.opens(AGENT)
    talks.get(id)?.writing(text)
    host.shows(id)
  }

  return { kind, asks }
}
