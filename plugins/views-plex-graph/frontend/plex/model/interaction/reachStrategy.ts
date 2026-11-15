/** How a node offers a gesture start: a mouse finds a handle by hovering over the node and presses it. */

/** The node a strategy is watching, in the only two terms it needs. */
interface ReachSite {
  readonly ready: () => boolean
  readonly reach: (event: PointerEvent) => void
}

export interface ReachStrategy {
  readonly hasHandle: boolean
  readonly listeners: (site: ReachSite) => Record<string, (event: PointerEvent) => void>
}

/** The handle strategy, for pointer and keyboard. */
export const byHandle: ReachStrategy = {
  hasHandle: true,
  listeners: () => ({}),
}
