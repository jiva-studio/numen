import type { PlacedEdge } from '../../edge'
import type { PlacedNode } from '../../node'
import type { PlexEdge } from '../../../../types'
import { routeEdge, type Routing } from './routing'

export interface RouteEdgesOptions {
  readonly routing: Routing
  readonly getOpacity?: (edge: PlexEdge) => number
}

export function routeEdges(
  edges: readonly PlexEdge[],
  byId: ReadonlyMap<string, PlacedNode>,
  routingOrOptions: Routing | RouteEdgesOptions,
): PlacedEdge[] {
  const routingOptions = 'tuning' in routingOrOptions ? routingOrOptions : routingOrOptions.routing
  const getOpacity =
    'tuning' in routingOrOptions ? () => 1 : (routingOrOptions.getOpacity ?? (() => 1))

  return edges
    .map((edge) => routeEdge(edge, byId, { routing: routingOptions, opacity: getOpacity(edge) }))
    .filter((edge): edge is PlacedEdge => edge !== null)
}
