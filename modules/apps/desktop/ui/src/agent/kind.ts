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

/** What an agent tab asks of the window it is drawn in. */
export interface Talking {
  /** The note the person is looking at, which is what a question is about. */
  looking(): string
  /** A source opened at stretches of its own text, the first of them in front. */
  opens(path: string, ...runs: readonly Run[]): void
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

  return { ...talk, asked, writing, send, opensTurn, followed }
}
