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
