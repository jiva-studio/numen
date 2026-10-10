/**
 * Helper functions for building and resolving link items in the Plex link inspector.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import type { DirectionalLinkItem, LinkDirectionMode } from '../types'

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
  related: { label?: string; reverseLabel?: string },
  isFocusA: boolean,
): { descAtoB: string; descBtoA: string } {
  const fwd = related.label ?? ''
  const rev = related.reverseLabel ?? ''
  return {
    descAtoB: isFocusA ? fwd : rev,
    descBtoA: isFocusA ? rev : fwd,
  }
}

export function createMutualInspectorLinks(
  pathA: string,
  pathB: string,
  related: { seat: string; label?: string; reverseLabel?: string },
  isFocusA: boolean,
  descriptions: Map<string, string>,
): DirectionalLinkItem[] {
  const baseSeat = (related.seat as PlexRelatedSeat) || 'jump'
  const roleAtoB = isFocusA ? baseSeat : getInverseSeat(baseSeat)
  const roleBtoA = getInverseSeat(roleAtoB)
  const defaults = getDefaultMutualDescriptions(related, isFocusA)

  const descAB = descriptions.get(`${pathA}->${pathB}`) ?? defaults.descAtoB
  const descBA = descriptions.get(`${pathB}->${pathA}`) ?? defaults.descBtoA

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

export function createSingleInspectorLink(
  pathA: string,
  pathB: string,
  related: { seat?: string; label?: string } | undefined,
  isFocusA: boolean,
  directions: Map<string, string>,
  descriptions: Map<string, string>,
): DirectionalLinkItem[] {
  const from = isFocusA ? pathA : pathB
  const to = isFocusA ? pathB : pathA
  const role: PlexRelatedSeat = (related?.seat as PlexRelatedSeat) || 'jump'
  const pairKey = [pathA, pathB].sort().join(' ')
  const savedDirection = directions.get(pairKey)
  const direction = resolveSingleDirection(savedDirection, role, isFocusA, pathA, pathB)
  const desc = related?.label ?? ''
  const savedDesc = descriptions.get(`${from}->${to}`) ?? desc

  return [
    {
      id: `${from}->${to}`,
      from,
      to,
      role,
      direction,
      description: savedDesc,
    },
  ]
}
