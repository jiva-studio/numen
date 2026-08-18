/**
 * What each tab of the window holds, and what letting go of one comes to.
 *
 * A tab holds a plex, a talk, or nothing yet. Which it is decides what is drawn
 * and what is released when it closes, and both are settled here so a test can
 * ask them without a browser. A tab holding a note is the window's own.
 */
import { computed, ref, shallowRef, type ComputedRef, type Ref } from 'vue'
import { closeTab, openTab, paneWithTab } from '@numen/ui'
import type { NodeId, PlexNeighbourhood, WorkspaceLayout } from '@numen/ui'
import { asPlex } from './plex'
import type { Conversation } from './conversation'
import type { Plexed } from './showing'
import { AGENT, BLANK, CONVERSATION, NOTE, PLEX, named, opening } from './workspace'

/** What one plex tab holds: where it is standing, and the picture it draws. */
export interface Held {
  readonly view: Plexed
  /** The plex reads one value, so what it is given changes when the vault does. */
  readonly picture: ComputedRef<PlexNeighbourhood | null>
}

/** What one agent tab holds: a talk of its own, and the question being written. */
export interface Talk extends Conversation {
  readonly asked: Ref<string>
}

/** What the window makes when a tab is told what it holds. */
export interface Makes {
  /** A plex of its own, standing where it is told or where the person is. */
  plex(at?: string): Plexed
  /** A talk of its own, answering under the name the agent hears it by. */
  talk(conversation: string): Conversation
  /** A note of its own, opened; where it is filed, or nothing when none was made. */
  note(): Promise<string>
}

export function holding(makes: Makes) {
  const plexes = shallowRef<ReadonlyMap<string, Held>>(new Map())
  const agents = shallowRef<ReadonlyMap<string, Talk>>(new Map())
  /** Tabs opened with nothing in them, each waiting to be told what it holds. */
  const blanks = ref<readonly string[]>([])
  /** The agent tab the person was last in. A question about a note goes there. */
  const talking = ref('')

  /** A plex of its own, and the tab it will stand in. */
  const plexTab = (at?: string): string => {
    const id = named(PLEX)
    const view = makes.plex(at)
    const picture = computed(() =>
      view.neighbourhood.value ? asPlex(view.neighbourhood.value) : null,
    )
    plexes.value = new Map(plexes.value).set(id, { view, picture })
    return id
  }

  /**
   * An agent of its own, and the tab it will talk in. The conversation carries
   * a name of its own: the agent keeps one conversation under each name it
   * hears, and a tab is a place on the screen.
   */
  const agentTab = (): string => {
    const id = named(AGENT)
    const talk = makes.talk(named(CONVERSATION))
    agents.value = new Map(agents.value).set(id, { asked: ref(''), ...talk })
    return id
  }

  const layout = ref<WorkspaceLayout>(opening(plexTab(), agentTab()))

  /** A tab opened empty, in the pane the plus was pressed in. */
  const blanked = (pane: NodeId) => {
    const id = named(BLANK)
    blanks.value = [...blanks.value, id]
    layout.value = openTab(layout.value, id, pane)
  }

  /** What a blank tab was told to be, put up where it stood. */
  const fills = (blank: string, tab: string) => {
    const pane = paneWithTab(layout.value.root, blank)?.id
    layout.value = openTab(layout.value, tab, pane ?? layout.value.focus)
    layout.value = closeTab(layout.value, blank)
    blanks.value = blanks.value.filter((one) => one !== blank)
  }

  /** A plex opened on a note, for a note to show and no plex to show it in. */
  const shows = (path: string) => {
    layout.value = openTab(layout.value, plexTab(path))
  }

  /** A blank tab told what to hold, and holding it where it stood. */
  const becomeIt = async (blank: string, what: string) => {
    if (what === PLEX) fills(blank, plexTab())
    else if (what === AGENT) fills(blank, agentTab())
    else if (what === NOTE) {
      const path = await makes.note()
      if (path) fills(blank, path)
    }
  }

  /** What is being written in one agent tab. */
  const writing = (id: string, text: string) => {
    const talk = agents.value.get(id)
    if (talk) talk.asked.value = text
  }

  /**
   * Something to ask, put in the agent the person was last in. With no agent
   * open, one is opened to carry it.
   */
  const askAbout = (text: string) => {
    const id = agents.value.has(talking.value) ? talking.value : agentTab()
    writing(id, text)
    layout.value = openTab(layout.value, id)
  }

  /** The tab now on screen: which plex the person is in, and which agent. */
  const shown = (id: string) => {
    plexes.value.get(id)?.view.looking()
    if (agents.value.has(id)) talking.value = id
  }

  /** One tab let go of, and the rest kept. */
  const without = <T,>(held: ReadonlyMap<string, T>, id: string): ReadonlyMap<string, T> => {
    const rest = new Map(held)
    rest.delete(id)
    return rest
  }

  /** A tab lets go of what it held, and says whether it held anything. */
  const shut = (id: string): boolean => {
    if (blanks.value.includes(id)) {
      blanks.value = blanks.value.filter((one) => one !== id)
      return true
    }

    const held = plexes.value.get(id)
    if (held) {
      held.view.close()
      plexes.value = without(plexes.value, id)
      return true
    }

    const talk = agents.value.get(id)
    if (talk) {
      talk.finish()
      agents.value = without(agents.value, id)
      if (talking.value === id) talking.value = ''
      return true
    }

    return false
  }

  /** The window is going, and nothing a tab holds outlives it. */
  const close = () => {
    for (const talk of agents.value.values()) talk.finish()
    for (const held of plexes.value.values()) held.view.close()
  }

  return {
    layout,
    plexes,
    agents,
    blanks,
    talking,
    plexTab,
    agentTab,
    blanked,
    fills,
    shows,
    becomeIt,
    writing,
    askAbout,
    shown,
    shut,
    close,
  }
}
