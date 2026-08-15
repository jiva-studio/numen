/**
 * Asking the application about the vault, and turning what comes back into
 * something the plex can draw.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it. What is written here is only the
 * translation from a vault into seats around a focus, which is the one thing
 * the plex must not know.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { Seat, Vault, type NeighbourhoodResponse } from '@numen/protocol'
import type { PlexNeighbourhood, PlexNode, PlexRole } from '@numen/ui'

export const vault = createClient(
  Vault,
  createConnectTransport({ baseUrl: window.location.origin }),
)

export type Neighbourhood = NeighbourhoodResponse

const seats: Record<Seat, PlexRole | null> = {
  [Seat.UNSPECIFIED]: null,
  [Seat.PARENT]: 'parent',
  [Seat.CHILD]: 'child',
  [Seat.JUMP]: 'jump',
  [Seat.SIBLING]: 'sibling',
}

/**
 * A note is addressed by the path it is filed under: every note has one, and a
 * note written outside the application carries no identifier.
 */
export function asPlex(neighbourhood: Neighbourhood): PlexNeighbourhood {
  const focus: PlexNode = {
    id: neighbourhood.focus?.path ?? '',
    label: neighbourhood.focus?.title ?? '',
    role: 'focus',
  }

  const nodes: PlexNode[] = [focus]
  const edges: { from: string; to: string }[] = []
  for (const related of neighbourhood.related) {
    const role = seats[related.seat]
    if (!role || !related.note) continue
    nodes.push({ id: related.note.path, label: related.note.title, role })
    edges.push({ from: focus.id, to: related.note.path })
  }
  return { nodes, edges }
}
