/**
 * Helper functions for building and resolving link items in the Plex link inspector.
 */
import type { PlexRelation } from '@numen/ui'
import type { DirectionalLinkItem, LinkDirectionMode } from '../types'

/**
 * Returns the opposite hierarchy relation for a relationship.
 * Note: 'sibling' is a graph placement relation derived from a shared parent, not an authored link role.
 */
export function getInverseRelation(relation: PlexRelation): PlexRelation {
  switch (relation) {
    case 'parent':
      return 'child'
    case 'child':
      return 'parent'
    case 'jump':
    case 'sibling':
      return relation
  }
}

/**
 * Resolves the initial link direction mode for a single link between two notes.
 * Single links default to undirected ('—') unless explicitly set to a direction.
 */
export function resolveSingleDirection(
  savedDirection: string | undefined,
  role: PlexRelation,
  isFocusA: boolean,
  pathA: string,
  pathB: string,
): LinkDirectionMode {
  if (savedDirection === 'undirected') return 'undirected'
  if (savedDirection) {
    return savedDirection === `${pathA}->${pathB}` ? 'forward' : 'reverse'
  }
  return 'undirected'
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
  related: { relation: string; directLabel?: string; label?: string; reverseLabel?: string },
  isFocusA: boolean,
  descriptions: Map<string, string>,
): DirectionalLinkItem[] {
  const baseRelation = (related.relation as PlexRelation) || 'jump'
  const roleAtoB = isFocusA ? baseRelation : getInverseRelation(baseRelation)
  const roleBtoA = getInverseRelation(roleAtoB)
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

function getSingleRole(baseRole: PlexRelation, isReverse: boolean): PlexRelation {
  return isReverse ? getInverseRelation(baseRole) : baseRole
}

export function createSingleInspectorLink(
  pathA: string,
  pathB: string,
  related: { relation?: string; directLabel?: string; label?: string } | undefined,
  isFocusA: boolean,
  directions: Map<string, string>,
  descriptions: Map<string, string>,
): DirectionalLinkItem[] {
  const baseRole = (related?.relation as PlexRelation) || 'jump'
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
  nodeAPath?: string,
  nodeBPath?: string,
): {
  direction: LinkDirectionMode
  from: string
  to: string
  role: PlexRelation
  removedLink?: { from: string; to: string; role?: PlexRelation } | undefined
} {
  const nextDirection: Record<LinkDirectionMode, LinkDirectionMode> = {
    undirected: 'forward',
    forward: 'reverse',
    reverse: 'undirected',
  }
  const next = nextDirection[row.direction]
  if (!nodeAPath || !nodeBPath) {
    return {
      direction: next,
      from: row.from,
      to: row.to,
      role: row.role,
      removedLink: undefined,
    }
  }

  const baseRole = row.from === nodeAPath ? row.role : getInverseRelation(row.role)
  if (next === 'reverse') {
    return {
      direction: next,
      from: nodeBPath,
      to: nodeAPath,
      role: getInverseRelation(baseRole),
      removedLink: undefined,
    }
  }

  return {
    direction: next,
    from: nodeAPath,
    to: nodeBPath,
    role: baseRole,
    removedLink: undefined,
  }
}

function pruneEmptyDualRows(
  rows: readonly DirectionalLinkItem[],
  initialLinkIds: Set<string>,
): {
  rows: DirectionalLinkItem[]
  removed: { from: string; to: string; role?: PlexRelation }[]
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
  alreadyRemoved: readonly { from: string; to: string; role?: PlexRelation }[],
): { from: string; to: string; role?: PlexRelation }[] {
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
  existingRemoved: readonly { from: string; to: string; role?: PlexRelation }[],
): {
  pairKey: string
  rows: DirectionalLinkItem[]
  removedLinks: { from: string; to: string; role?: PlexRelation }[]
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

export interface PopoverPlacementOptions {
  readonly at: { readonly x: number; readonly y: number }
  readonly popoverSize: { readonly width: number; readonly height: number }
  readonly containerSize: { readonly width: number; readonly height: number }
  readonly margin?: number
  readonly gap?: number
}

export interface PopoverPlacementResult {
  readonly left: number
  readonly top: number
}

function getLeftPosition(atX: number, width: number, containerW: number, margin: number): number {
  const idealLeft = atX - width / 2
  const maxLeft = Math.max(margin, containerW - width - margin)
  return Math.max(margin, Math.min(idealLeft, maxLeft))
}

function getTopPosition(
  atY: number,
  height: number,
  containerH: number,
  margin: number,
  gap: number,
): number {
  const aboveTop = atY - height - gap
  if (aboveTop >= margin) return aboveTop

  const belowTop = atY + gap
  const maxTop = Math.max(margin, containerH - height - margin)
  if (belowTop + height <= containerH - margin) return belowTop

  const clampedAbove = Math.max(margin, Math.min(aboveTop, maxTop))
  const clampedBelow = Math.max(margin, Math.min(belowTop, maxTop))
  return atY < containerH / 2 ? clampedBelow : clampedAbove
}

export function resolvePopoverPosition({
  at,
  popoverSize,
  containerSize,
  margin = 12,
  gap = 12,
}: PopoverPlacementOptions): PopoverPlacementResult {
  const width = popoverSize.width || 320
  const height = popoverSize.height || 160
  const containerW =
    containerSize.width || (typeof window !== 'undefined' ? window.innerWidth : 800)
  const containerH =
    containerSize.height || (typeof window !== 'undefined' ? window.innerHeight : 600)

  return {
    left: Math.round(getLeftPosition(at.x, width, containerW, margin)),
    top: Math.round(getTopPosition(at.y, height, containerH, margin, gap)),
  }
}
