/**
 * What one agent tab holds: a talk of its own, and what pressing anything in
 * it comes to.
 *
 * What the person has open is the window's to report, and it says so as it
 * changes. A line about work names a place in a source and opens it; a link
 * inside an answer names a place in a source or a note, and opens either.
 */
import { computed, ref, shallowRef, watch } from 'vue'
import { pointsAtNote, wikilinksIn, type Conversation, type Turn } from '@numen/ui'
import { same, spotOf, spotsIn } from './places'
import type { Span } from '../shared/core'
import type { Kind, WindowHandle } from '../shared/tabs/windowTabs'
import { AGENT, shortened } from '../shared/tabs/workspace'
import AgentTab from './AgentTab.vue'
import { WORDS as words } from './words'

/** What an agent tab asks of the window it is drawn in. */
export interface AgentTabDeps {
  /** A source opened at stretches of its own text, the first of them in front. */
  opens(path: string, ...spans: readonly Span[]): void
  /** A note opened in a tab beside the pane the person is in. */
  beside(path: string): void
  /**
   * Where each of those addresses lands, by the address it was asked about.
   * One that reaches nothing is absent.
   */
  resolve(written: readonly string[]): Promise<ReadonlyMap<string, string>>
  /** Why the agent cannot be reached, which the tab says where its answers stand. */
  unreachable(): string
}

/** What one agent tab holds. */
export type AgentTabState = ReturnType<typeof talking>

/** The note a talk is about, under the name the window calls it by. */
export interface NoteRef {
  readonly path: string
  readonly title: string
}

export function talking(talk: Conversation, deps: AgentTabDeps) {
  /** The question being written, until it is sent. */
  const asked = ref('')

  const writing = (text: string) => {
    asked.value = text
  }

  /** A question sent. What the person has open the agent reads for itself. */
  const send = (text: string) => {
    asked.value = ''
    void talk.ask(text, '')
  }

  /** A line about work pressed: the place that call was on is put in front. */
  const opensTurn = (turn: Turn) => {
    const at = talk.place(turn.id)
    if (at) deps.opens(at.path, at.span)
  }

  /**
   * Where each address an answer points at lands. An address that reaches
   * nothing is held as the empty path, which is what draws the link as not
   * resolving.
   */
  const landed = shallowRef<ReadonlyMap<string, string>>(new Map())
  const asking = new Set<string>()

  /** Every address an answer points at, asked once each as they arrive. */
  const asks = (addresses: readonly string[]) => {
    const fresh = addresses.filter((one) => !asking.has(one))
    if (!fresh.length) return
    for (const one of fresh) asking.add(one)
    void deps.resolve(fresh).then((found) => {
      const next = new Map(landed.value)
      for (const one of fresh) next.set(one, found.get(one) ?? '')
      landed.value = next
    })
  }

  const addressesIn = (text: string) => wikilinksIn(text).map((one) => one.address)

  watch(
    talk.turns,
    (all) => asks(all.flatMap((turn) => addressesIn(turn.text))),
    { deep: true },
  )

  /** The turns as the thread draws them, each saying which of its links reach nothing. */
  const turns = computed<Turn[]>(() =>
    talk.turns.value.map((turn) => {
      const unresolved = addressesIn(turn.text).filter(
        (address) => landed.value.get(address) === '',
      )
      return unresolved.length ? { ...turn, unresolved } : turn
    }),
  )

  /**
   * A link inside an answer pressed. One naming a place in the vault opens it,
   * and the other places that answer names in the same document are lit with
   * it. One naming a note opens that note beside what the person is looking
   * at. Any other link is left to whatever would follow it.
   */
  const followed = (turn: Turn, href: string, press: MouseEvent) => {
    const here = spotOf(href)
    if (here) {
      press.preventDefault()
      const named = spotsIn(turn.text).filter((spot) => spot.path === here.path && !same(spot, here))
      deps.opens(
        here.path,
        ...[here, ...named].map(({ start, length }) => ({ from: start, to: start + length })),
      )
      return
    }
    if (!pointsAtNote(href)) return
    const path = landed.value.get(href)
    if (path) deps.beside(path)
  }

  return {
    ...talk,
    turns,
    asked,
    writing,
    send,
    opensTurn,
    followed,
    unreachable: deps.unreachable,
  }
}

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
  const kind: Kind<AgentTabState> = {
    kind: AGENT,
    opens,
    called: (state) =>
      shortened(state.turns.value.find((turn) => turn.voice === 'asked')?.text ?? '') || words.agent,
    draws: AgentTab,
    shuts: (state) => {
      state.finish()
      return true
    },
    at: () => about(),
  }

  /** Something to ask, put in the agent the person was last in and put in front. */
  const asks = async (text: string) => {
    const id = handle.last<AgentTabState>(AGENT)?.id ?? (await handle.opens(AGENT))
    handle.holds<AgentTabState>(AGENT, id)?.writing(text)
    handle.shows(id)
  }

  return { kind, asks }
}
