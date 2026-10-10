/**
 * Helper functions for building and resolving link items in the Plex link inspector.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import type { DirectionalLinkItem, LinkDirectionMode } from '../types'

/**
 * Returns the opposite hierarchy seat for a relationship.
 * Note: 'sibling' is a graph placement seat derived from a shared parent, not an authored link role.
 */
export function getInverseSeat(seat: PlexRelatedSeat): PlexRelatedSeat {
  switch (seat) {
    case 'parent':
      return 'child'
    case 'child':
      return 'parent'
    case 'jump':
    case 'sibling':
      return seat
  }
}

/**
 * Resolves the initial link direction mode for a single link between two notes.
 * Jump links default to undirected ('—') while hierarchical parent/child links default to directional.
 */
export function resolveSingleDirection(
  savedDirection: string | undefined,
  role: PlexRelatedSeat,
  isFocusA: boolean,
  pathA: string,
  pathB: string,
): LinkDirectionMode {
  if (savedDirection === 'undirected') return 'undirected'
  if (savedDirection) {
    return savedDirection === `${pathA}->${pathB}` ? 'forward' : 'reverse'
  }
  if (role === 'jump') return 'undirected'
  return isFocusA ? 'forward' : 'reverse'
}

function getDefaultMutualDescriptions(
  related: { directLabel?: string; label?: string; reverseLabel?: string },
  isFocusA: boolean,
): { directDesc: string; reverseDesc: string } {
  const direct = related.directLabel ?? related.label ?? ''
  const reverse = related.reverseLabel ?? ''
  return {
    directDesc: isFocusA ? direct : reverse,
    reverseDesc: isFocusA ? reverse : direct,
  }
}

export function createMutualInspectorLinks(
  pathA: string,
  pathB: string,
  related: { seat: string; directLabel?: string; label?: string; reverseLabel?: string },
  isFocusA: boolean,
  descriptions: Map<string, string>,
): DirectionalLinkItem[] {
  const baseSeat = (related.seat as PlexRelatedSeat) || 'jump'
  const roleAtoB = isFocusA ? baseSeat : getInverseSeat(baseSeat)
  const roleBtoA = getInverseSeat(roleAtoB)
  const defaults = getDefaultMutualDescriptions(related, isFocusA)

  const descAB = descriptions.get(`${pathA}->${pathB}`) ?? defaults.directDesc
  const descBA = descriptions.get(`${pathB}->${pathA}`) ?? defaults.reverseDesc

  return [
    {
      id: `${pathA}->${pathB}`,
      from: pathA,
      to: pathB,
      role: roleAtoB,
      direction: 'forward',
      description: descAB,
    },
    {
      id: `${pathB}->${pathA}`,
      from: pathB,
      to: pathA,
      role: roleBtoA,
      direction: 'reverse',
      description: descBA,
    },
  ]
}

function getSingleRole(baseRole: PlexRelatedSeat, isReverse: boolean): PlexRelatedSeat {
  return isReverse ? getInverseSeat(baseRole) : baseRole
}

export function createSingleInspectorLink(
  pathA: string,
  pathB: string,
  related: { seat?: string; directLabel?: string; label?: string } | undefined,
  isFocusA: boolean,
  directions: Map<string, string>,
  descriptions: Map<string, string>,
): DirectionalLinkItem[] {
  const baseRole = (related?.seat as PlexRelatedSeat) || 'jump'
  const pairKey = [pathA, pathB].sort().join(' ')
  const direction = resolveSingleDirection(
    directions.get(pairKey),
    baseRole,
    isFocusA,
    pathA,
    pathB,
  )

  const isReverse = direction === 'reverse'
  const from = isReverse ? pathB : pathA
  const to = isReverse ? pathA : pathB
  const role = getSingleRole(baseRole, isReverse)
  const desc = related?.directLabel ?? related?.label ?? ''
  const description = descriptions.get(`${from}->${to}`) ?? desc

  return [{ id: `${from}->${to}`, from, to, role, direction, description }]
}

export function resolveInspectorDirectionToggle(
  row: DirectionalLinkItem,
  nodeAPath: string,
  nodeBPath: string,
  initialLinkIds: Set<string>,
): {
  direction: LinkDirectionMode
  from: string
  to: string
  role: PlexRelatedSeat
  removedLink?: { from: string; to: string; role?: PlexRelatedSeat } | undefined
} {
  const nextDirection: Record<LinkDirectionMode, LinkDirectionMode> = {
    undirected: 'forward',
    forward: 'reverse',
    reverse: 'undirected',
  }
  const next = nextDirection[row.direction]
  const baseRole = row.from === nodeAPath ? row.role : getInverseSeat(row.role)

  if (next === 'reverse') {
    const removedLink =
      initialLinkIds.has(row.id) && row.from === nodeAPath
        ? { from: nodeAPath, to: nodeBPath, role: baseRole }
        : undefined
    return {
      direction: next,
      from: nodeBPath,
      to: nodeAPath,
      role: getInverseSeat(baseRole),
      removedLink,
    }
  }

  const removedLink =
    initialLinkIds.has(row.id) && row.from === nodeBPath
      ? { from: nodeBPath, to: nodeAPath, role: getInverseSeat(baseRole) }
      : undefined
  return {
    direction: next,
    from: nodeAPath,
    to: nodeBPath,
    role: baseRole,
    removedLink,
  }
}

function pruneEmptyDualRows(
  rows: readonly DirectionalLinkItem[],
  initialLinkIds: Set<string>,
): {
  rows: DirectionalLinkItem[]
  removed: { from: string; to: string; role?: PlexRelatedSeat }[]
} {
  if (rows.length !== 2) return { rows: [...rows], removed: [] }
  const r0Blank = !rows[0]!.description.trim()
  const r1Blank = !rows[1]!.description.trim()

  if (r0Blank && !r1Blank) {
    const r0 = rows[0]!
    const removed = initialLinkIds.has(r0.id) ? [{ from: r0.from, to: r0.to, role: r0.role }] : []
    return { rows: [rows[1]!], removed }
  }
  if (r1Blank && !r0Blank) {
    const r1 = rows[1]!
    const removed = initialLinkIds.has(r1.id) ? [{ from: r1.from, to: r1.to, role: r1.role }] : []
    return { rows: [rows[0]!], removed }
  }
  return { rows: [...rows], removed: [] }
}

function collectOmittedInitialLinks(
  initialLinks: readonly DirectionalLinkItem[],
  finalRows: readonly DirectionalLinkItem[],
  alreadyRemoved: readonly { from: string; to: string; role?: PlexRelatedSeat }[],
): { from: string; to: string; role?: PlexRelatedSeat }[] {
  const isPresent = (link: DirectionalLinkItem) => finalRows.some((r) => r.id === link.id)
  const isRemoved = (link: DirectionalLinkItem) =>
    alreadyRemoved.some((r) => r.from === link.from && r.to === link.to && r.role === link.role)

  return initialLinks
    .filter((link) => !isPresent(link) && !isRemoved(link))
    .map((link) => ({ from: link.from, to: link.to, role: link.role }))
}

export function resolveInspectorDonePayload(
  localRows: readonly DirectionalLinkItem[],
  initialLinks: readonly DirectionalLinkItem[],
  initialLinkIds: Set<string>,
  pairKey: string,
  existingRemoved: readonly { from: string; to: string; role?: PlexRelatedSeat }[],
): {
  pairKey: string
  rows: DirectionalLinkItem[]
  removedLinks: { from: string; to: string; role?: PlexRelatedSeat }[]
} {
  const pruned = pruneEmptyDualRows(localRows, initialLinkIds)
  const allRemoved = [...existingRemoved, ...pruned.removed]
  const omitted = collectOmittedInitialLinks(initialLinks, pruned.rows, allRemoved)

  return {
    pairKey,
    rows: pruned.rows,
    removedLinks: [...allRemoved, ...omitted],
  }
}
