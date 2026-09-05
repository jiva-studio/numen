/**
 * A node asked for on its own: how it is asked for, and where it is then
 * drawn. A hand asks with a second click, which the browser reports as one
 * event; a finger's two taps are counted here.
 *
 * Which of the two a plex offers is the caller's, so neither is written into
 * the node.
 */
import { onScopeDispose, ref } from 'vue'

/**
 * Where a node asked for is to be drawn, declared once. Everything that
 * varies with it — the modifier that asks for it, and what a caller does about
 * it — is derived from this table.
 */
export interface ShowingDescriptor {
  /** Whether the modifier is held while asking for this one. */
  readonly modified: boolean
}

export const SHOWINGS = {
  here: { modified: false },
  beside: { modified: true },
} as const satisfies Record<string, ShowingDescriptor>

/** Where a node asked for is to be drawn: where the reader is, or beside it. */
export type PlexShowing = keyof typeof SHOWINGS

export const SHOWINGS_ALL = Object.keys(SHOWINGS) as readonly PlexShowing[]

/** One name per modifier, taken from the table. */
const BY_MODIFIER = Object.fromEntries(
  SHOWINGS_ALL.map((showing) => [SHOWINGS[showing].modified, showing]),
) as Readonly<Record<`${boolean}`, PlexShowing>>

/**
 * Which one was asked for, given the modifier held at the time. It is read
 * once, where the gesture arrives, and what travels on is the answer.
 */
export const showingOf = (modified: boolean): PlexShowing => BY_MODIFIER[`${modified}`]

/** How long the second tap has to arrive in, in milliseconds. */
export const TAP = 300

/** How far it may land from the first, in pixels. */
export const APART = 24

/** The node a strategy is watching, in the only two terms it needs. */
export interface ShowingSite {
  /** Whether this node answers being asked for at all. */
  readonly ready: () => boolean
  /** It was asked for. The modifier says where it is to be drawn. */
  readonly show: (modified: boolean) => void
}

export interface Showing {
  /** Whether the node answers the browser's own second click. */
  readonly doubleClick: boolean
  /** What the node listens for besides. Called once, inside the node's scope. */
  readonly listeners: (site: ShowingSite) => Record<string, (event: PointerEvent) => void>
}

/** The second click, as the browser counts it. */
export const byDoubleClick: Showing = {
  doubleClick: true,
  listeners: () => ({}),
}

/** Two taps, counted here. Milliseconds, if the wait is to be another. */
export const byDoubleTap = (within: number = TAP): Showing => ({
  doubleClick: false,
  listeners: (site) => {
    const first = ref<{ timer: number; x: number; y: number } | null>(null)

    const forget = () => {
      if (!first.value) return
      window.clearTimeout(first.value.timer)
      first.value = null
    }

    onScopeDispose(forget)

    return {
      pointerup: (event: PointerEvent) => {
        if (event.pointerType === 'mouse' || !site.ready()) return
        const tap = first.value
        if (tap && Math.hypot(event.clientX - tap.x, event.clientY - tap.y) <= APART) {
          forget()
          site.show(false)
          return
        }
        forget()
        first.value = {
          x: event.clientX,
          y: event.clientY,
          timer: window.setTimeout(forget, within),
        }
      },
    }
  },
})

/**
 * Two sets of listeners on one element. A key both name is called for both, in
 * the order they were given.
 */
export function joined(
  ...held: Record<string, (event: PointerEvent) => void>[]
): Record<string, (event: PointerEvent) => void> {
  const all: Record<string, ((event: PointerEvent) => void)[]> = {}
  for (const one of held) {
    for (const [name, listener] of Object.entries(one)) {
      ;(all[name] ??= []).push(listener)
    }
  }
  return Object.fromEntries(
    Object.entries(all).map(([name, listeners]) => [
      name,
      (event: PointerEvent) => listeners.forEach((listener) => listener(event)),
    ]),
  )
}
