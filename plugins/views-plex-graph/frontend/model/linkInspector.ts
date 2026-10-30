/**
 * The links between two notes as the link inspector shows them, and the user's edits to its rows.
 */
import { measureWindow } from '@numen/ui'
import type { PlexRelation } from '../plex'
import type { Neighbourhood, DirectionalLinkItem } from '../types'

type Neighbour = Neighbourhood['related'][number]

/** The opposite hierarchy relation; a sibling is placed by a shared parent and is its own opposite. */
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

function createLinkItem(
  from: string,
  to: string,
  ...[role, label, firstPath]: [role: PlexRelation, label: string, firstPath: string]
): DirectionalLinkItem {
  return {
    id: `${from}->${to}`,
    from,
    to,
    role,
    direction: from === firstPath ? 'forward' : 'reverse',
    label,
  }
}

/** The neighbour that the line between two drawn notes represents. */
export function findPairNeighbour(
  neighbourhood: Neighbourhood | null,
  pathA: string,
  pathB: string,
): Neighbour | undefined {
  if (!neighbourhood) return undefined
  const focus = neighbourhood.focus.path
  if (pathA === focus || pathB === focus) {
    const other = pathA === focus ? pathB : pathA
    return neighbourhood.related.find((one) => one.path === other)
  }
  return neighbourhood.related.find(
    (one) =>
      one.relation === 'sibling' &&
      [pathA, pathB].includes(one.path) &&
      [pathA, pathB].includes(one.through),
  )
}

/** The links the core reports between two drawn notes, the one written in `pathA` first. */
export function createInspectorLinks(
  neighbourhood: Neighbourhood | null,
  pathA: string,
  pathB: string,
): DirectionalLinkItem[] {
  const related = findPairNeighbour(neighbourhood, pathA, pathB)
  if (!neighbourhood || !related) return []

  if (related.relation === 'sibling') {
    return [createLinkItem(related.path, related.through, 'parent', related.label, pathA)]
  }

  const focus = neighbourhood.focus.path
  const links = [createLinkItem(focus, related.path, related.relation, related.label, pathA)]
  if (related.isMutual) {
    const reverse = related.reverseLabel ?? ''
    links.push(
      createLinkItem(related.path, focus, getInverseRelation(related.relation), reverse, pathA),
    )
  }
  return focus === pathA ? links : links.reverse()
}

/** The role a link written in `from` takes, going by the first link the pair had. */
function getRoleFrom(initial: readonly DirectionalLinkItem[], from: string): PlexRelation {
  const first = initial[0]
  if (!first) return 'jump'
  return first.from === from ? first.role : getInverseRelation(first.role)
}

/** The only link flipped to run the other way. Two links are not flipped. */
export function toggleLinkDirection(
  rows: readonly DirectionalLinkItem[],
  firstPath: string,
): DirectionalLinkItem[] {
  if (rows.length !== 1) return [...rows]
  const row = rows[0]!
  return [createLinkItem(row.to, row.from, getInverseRelation(row.role), row.label, firstPath)]
}

/** The rows with the label of one replaced. */
export function setLinkLabel(
  rows: readonly DirectionalLinkItem[],
  id: string,
  label: string,
): DirectionalLinkItem[] {
  return rows.map((row) => (row.id === id ? { ...row, label } : row))
}

/** The rows without one. */
export function removeLink(
  rows: readonly DirectionalLinkItem[],
  id: string,
): DirectionalLinkItem[] {
  return rows.filter((row) => row.id !== id)
}

/** The rows with the link back added; two rows are the most there are. */
export function addReverseLink(
  rows: readonly DirectionalLinkItem[],
  initial: readonly DirectionalLinkItem[],
  ...[firstPath, secondPath]: [firstPath: string, secondPath: string]
): DirectionalLinkItem[] {
  if (rows.length >= 2) return [...rows]
  const first = rows[0]
  if (!first) {
    const role = getRoleFrom(initial, firstPath)
    return [createLinkItem(firstPath, secondPath, role, '', firstPath)]
  }
  const back = createLinkItem(first.to, first.from, getInverseRelation(first.role), '', firstPath)
  const forward = createLinkItem(first.from, first.to, first.role, first.label, firstPath)
  return first.from === firstPath ? [forward, back] : [back, forward]
}

/** The links to save: of two links, one with nothing written is left out. */
export function getLinksToSave(rows: readonly DirectionalLinkItem[]): DirectionalLinkItem[] {
  if (rows.length !== 2) return [...rows]
  const [one, other] = rows as [DirectionalLinkItem, DirectionalLinkItem]
  const isOneBlank = !one.label.trim()
  const isOtherBlank = !other.label.trim()
  if (isOneBlank && !isOtherBlank) return [other]
  if (isOtherBlank && !isOneBlank) return [one]
  return [...rows]
}

/** Whether two sets of links say the same thing. */
export function areLinksEqual(
  one: readonly DirectionalLinkItem[],
  other: readonly DirectionalLinkItem[],
): boolean {
  if (one.length !== other.length) return false
  return one.every((link) =>
    other.some(
      (match) =>
        match.from === link.from &&
        match.to === link.to &&
        match.role === link.role &&
        match.label === link.label,
    ),
  )
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

function getLeftPosition(
  atX: number,
  width: number,
  ...[containerW, margin]: [containerW: number, margin: number]
): number {
  const idealLeft = atX - width / 2
  const maxLeft = Math.max(margin, containerW - width - margin)
  return Math.max(margin, Math.min(idealLeft, maxLeft))
}

function getTopPosition(
  atY: number,
  height: number,
  ...[containerH, margin, gap]: [containerH: number, margin: number, gap: number]
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
  at: point,
  popoverSize,
  containerSize,
  margin = 12,
  gap = 12,
}: PopoverPlacementOptions): PopoverPlacementResult {
  const width = popoverSize.width || 320
  const height = popoverSize.height || 160
  const inner = measureWindow()
  const containerW = containerSize.width || inner.width
  const containerH = containerSize.height || inner.height

  return {
    left: Math.round(getLeftPosition(point.x, width, containerW, margin)),
    top: Math.round(getTopPosition(point.y, height, containerH, margin, gap)),
  }
}
