/**
 * Opening a node: how the request is made and where the node is shown. A mouse
 * opens with a double click, which the browser reports as one event.
 */
import type { PlexDestination } from '../../../types'

/**
 * Where an opened node is shown, declared once. The modifier that selects the
 * destination and the caller's behavior are derived from this table.
 */
interface DestinationDescriptor {
  readonly hasModifier: boolean
}

const DESTINATIONS = {
  current: { hasModifier: false },
  split: { hasModifier: true },
} as const satisfies Record<PlexDestination, DestinationDescriptor>

const DESTINATIONS_ALL = Object.keys(DESTINATIONS) as readonly PlexDestination[]

/** One name per modifier, taken from the table. */
const BY_MODIFIER = Object.fromEntries(
  DESTINATIONS_ALL.map((one) => [DESTINATIONS[one].hasModifier, one]),
) as Readonly<Record<`${boolean}`, PlexDestination>>

/**
 * The destination selected by the modifier state. It is read once, where the
 * gesture arrives, and the result is passed on.
 */
export const getDestination = (hasModifier: boolean): PlexDestination =>
  BY_MODIFIER[`${hasModifier}`]

/** The node a strategy is watching, in the only two terms it needs. */
interface ShowSite {
  readonly ready: () => boolean
  readonly show: (hasModifier: boolean) => void
}

export interface ShowStrategy {
  readonly isDoubleClick: boolean
  readonly listeners: (site: ShowSite) => Record<string, (event: PointerEvent) => void>
}

/** The second click, as the browser counts it. */
export const byDoubleClick: ShowStrategy = {
  isDoubleClick: true,
  listeners: () => ({}),
}

/**
 * Two sets of listeners on one element. A key both name is called for both, in
 * the order they were given.
 */
export function mergeListeners(
  ...target: Record<string, (event: PointerEvent) => void>[]
): Record<string, (event: PointerEvent) => void> {
  const all: Record<string, ((event: PointerEvent) => void)[]> = {}
  for (const one of target) {
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
