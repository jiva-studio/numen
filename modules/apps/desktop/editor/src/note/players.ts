/**
 * What link notes play, held outside the tabs that draw them.
 *
 * A frame put into the page again loads the site again, and a tab dragged to
 * another pane is drawn again from nothing. So the element stands in a layer of
 * its own and the tab says where to draw it, and a tab moved, hidden or shown
 * plays on from where it stood.
 *
 * One player to a tab. Two panes side by side each play what their own tab
 * points at.
 */
import { ref, type Ref } from 'vue'

/** Where a tab draws its player, in the window's own coordinates. */
export interface Box {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

/** One player, as the layer that draws it reads it. */
export interface Playing {
  readonly id: string
  /** What is framed, and nothing where a copy is played instead. */
  readonly embed: string
  /** A copy on this disk, and nothing where there is none. */
  readonly copy: string
  /** What it is called. */
  readonly title: string
  /** Where the tab draws it, and nothing while no tab draws it. */
  readonly box: Box | null
}

/** What a tab hands over when it draws its player. */
export interface Drawn {
  readonly id: string
  readonly embed: string
  readonly copy: string
  readonly title: string
  /** Where the tab draws it, and nothing while the tab stands behind another. */
  readonly box: Box | null
}

/** The element a player is played through. */
export type Played = HTMLIFrameElement | HTMLVideoElement

/** The players a window holds, as a tab and the layer drawing them read them. */
export interface Players {
  /** Every player the window holds, for the layer to draw. */
  readonly all: Ref<readonly Playing[]>
  /** A tab draws its player at this box. */
  draws(one: Drawn): void
  /** No tab is drawing it: it is kept, and drawn nowhere. */
  hides(id: string): void
  /** The tab is gone, and its player with it. */
  drops(id: string): void
  /** The element one is played through, as the layer hands it over. */
  drew(id: string, element: Played | null): void
  /** Play from a moment, in milliseconds. */
  seeks(id: string, ms: number): void
}

export function players(): Players {
  const all = ref<readonly Playing[]>([])
  const elements = new Map<string, Played>()

  const at = (id: string): number => all.value.findIndex((one) => one.id === id)

  const put = (one: Playing): void => {
    const found = at(one.id)
    const kept = [...all.value]
    if (found < 0) kept.push(one)
    else kept[found] = one
    all.value = kept
  }

  return {
    all,
    draws: (one) => {
      const found = at(one.id)
      const held = found < 0 ? null : all.value[found]
      if (
        held &&
        held.embed === one.embed &&
        held.copy === one.copy &&
        held.title === one.title &&
        held.box?.x === one.box?.x &&
        held.box?.y === one.box?.y &&
        held.box?.width === one.box?.width &&
        held.box?.height === one.box?.height
      ) {
        return
      }
      put(one)
    },
    hides: (id) => {
      const found = at(id)
      const held = found < 0 ? null : all.value[found]
      if (!held || held.box === null) return
      put({ ...held, box: null })
    },
    drops: (id) => {
      elements.delete(id)
      const found = at(id)
      if (found < 0) return
      all.value = all.value.filter((one) => one.id !== id)
    },
    drew: (id, element) => {
      if (element) elements.set(id, element)
      else elements.delete(id)
    },
    seeks: (id, ms) => {
      const element = elements.get(id)
      if (!element) return
      if (element instanceof HTMLVideoElement) {
        element.currentTime = ms / 1000
        void element.play()
        return
      }
      const held = all.value[at(id)]
      if (!held?.embed) return
      element.contentWindow?.postMessage(
        JSON.stringify({ event: 'command', func: 'seekTo', args: [ms / 1000, true] }),
        new URL(held.embed).origin,
      )
    },
  }
}

/** The players of this window. A test makes its own. */
export const held: Players = players()
