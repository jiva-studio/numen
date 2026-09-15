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
export interface DestinationDescriptor {
  /** Whether the modifier is held while asking for this one. */
  readonly modified: boolean
}

export const DESTINATIONS = {
  here: { modified: false },
  beside: { modified: true },
} as const satisfies Record<string, DestinationDescriptor>

/** Where a node asked for is to be drawn: where the reader is, or beside it. */
export type PlexDestination = keyof typeof DESTINATIONS

export const DESTINATIONS_ALL = Object.keys(DESTINATIONS) as readonly PlexDestination[]

/** One name per modifier, taken from the table. */
const BY_MODIFIER = Object.fromEntries(
  DESTINATIONS_ALL.map((one) => [DESTINATIONS[one].modified, one]),
) as Readonly<Record<`${boolean}`, PlexDestination>>

/**
 * Which one was asked for, given the modifier held at the time. It is read
 * once, where the gesture arrives, and what travels on is the answer.
 */
export const getDestination = (hasModifier: boolean): PlexDestination =>
  BY_MODIFIER[`${hasModifier}`]

/** How long the second tap has to arrive in, in milliseconds. */
export const TAP = 300

/** How far it may land from the first, in pixels. */
export const APART = 24

/** The node a strategy is watching, in the only two terms it needs. */
export interface ShowSite {
  /** Whether this node answers being asked for at all. */
  readonly ready: () => boolean
  /** It was asked for. The modifier says where it is to be drawn. */
  readonly show: (modified: boolean) => void
}

export interface ShowStrategy {
  /** Whether the node answers the browser's own second click. */
  readonly doubleClick: boolean
  /** What the node listens for besides. Called once, inside the node's scope. */
  readonly listeners: (site: ShowSite) => Record<string, (event: PointerEvent) => void>
}

/** The second click, as the browser counts it. */
export const byDoubleClick: ShowStrategy = {
  doubleClick: true,
  listeners: () => ({}),
}

/** Two taps, counted here. Milliseconds, if the wait is to be another. */
export const byDoubleTap = (within: number = TAP): ShowStrategy => ({
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
export function mergeListeners(
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
