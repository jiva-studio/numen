import type { Neighbor, Neighborhood, Relation } from './neighborhood'

export const PLEX_WIDTH = 630
export const PLEX_HEIGHT = 338

const NODE_HEIGHT = 40
/** The widest a title is drawn, in the units of the view box, before it is cut short. */
const MOST_TEXT = 112
const PADDING = 28
const ROW_SPACING = 155
const FOCUS = { x: PLEX_WIDTH / 2, y: 169 }
const ROW_Y = { parent: 38, child: 299 } as const
const COLUMN_X = { jump: 111, sibling: 515 } as const
const COLUMN_SPACING = 58

export interface PlexNode {
  readonly path: string
  readonly text: string
  readonly relation: Relation | 'focus'
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

/**
 * A curve from the note it leaves to the note its arrow points at. `labelPath` is the same curve drawn from
 * left to right, which the label is written along so that it reads upright.
 */
export interface PlexEdge {
  readonly id: string
  readonly path: string
  readonly labelPath: string
  readonly label: string
  readonly relation: Relation
}

export interface PlexLayout {
  readonly nodes: readonly PlexNode[]
  readonly edges: readonly PlexEdge[]
}

const NARROW = new Set([..."iljtfrI.,:; '()[]"])
const WIDE = new Set([...'mwMW@'])

const getLetterWidth = (letter: string): number => (NARROW.has(letter) ? 3.6 : WIDE.has(letter) ? 11 : 7.4)

function getText(title: string): string {
  let width = 0
  for (const [index, letter] of [...title].entries()) {
    width += getLetterWidth(letter)
    if (width > MOST_TEXT) return `${title.slice(0, index)}…`
  }
  return title
}

function getWidth(text: string, isFocus: boolean): number {
  const width = [...text].reduce((sum, letter) => sum + getLetterWidth(letter), 0) + PADDING
  return Math.max(isFocus ? 100 : 70, width)
}

function createNode(path: string, title: string, relation: PlexNode['relation'], at: { x: number; y: number }): PlexNode {
  const text = getText(title)
  return { path, text, relation, x: at.x, y: at.y, width: getWidth(text, relation === 'focus'), height: NODE_HEIGHT }
}

const spread = (count: number, spacing: number, middle: number): number[] =>
  Array.from({ length: count }, (_, index) => middle + (index - (count - 1) / 2) * spacing)

function placeNeighbors(related: readonly Neighbor[]): PlexNode[] {
  const nodes: PlexNode[] = []
  for (const relation of ['parent', 'child'] as const) {
    const group = related.filter((one) => one.relation === relation)
    const spacing = Math.min(ROW_SPACING, (PLEX_WIDTH - 60) / Math.max(group.length, 1))
    spread(group.length, spacing, FOCUS.x).forEach((x, index) => {
      const one = group[index]
      if (one) nodes.push(createNode(one.path, one.title, relation, { x, y: ROW_Y[relation] }))
    })
  }
  for (const relation of ['jump', 'sibling'] as const) {
    const group = related.filter((one) => one.relation === relation)
    spread(group.length, COLUMN_SPACING, FOCUS.y).forEach((y, index) => {
      const one = group[index]
      if (one) nodes.push(createNode(one.path, one.title, relation, { x: COLUMN_X[relation], y }))
    })
  }
  return nodes
}

const getCurve = (from: { x: number; y: number }, to: { x: number; y: number }, isVertical: boolean): string => {
  const mid = isVertical ? { x: from.x, y: (from.y + to.y) / 2 } : { x: (from.x + to.x) / 2, y: from.y }
  const mirror = isVertical ? { x: to.x, y: mid.y } : { x: mid.x, y: to.y }
  return `M ${from.x} ${from.y} C ${mid.x} ${mid.y}, ${mirror.x} ${mirror.y}, ${to.x} ${to.y}`
}

function createEdge(one: Neighbor, node: PlexNode, source: PlexNode): PlexEdge {
  const isVertical = one.relation !== 'jump'
  const down = node.y > source.y
  const from = isVertical
    ? { x: source.x, y: source.y + (down ? source.height / 2 : -source.height / 2) }
    : { x: source.x - source.width / 2, y: source.y }
  const to = isVertical
    ? { x: node.x, y: node.y + (down ? -node.height / 2 : node.height / 2) }
    : { x: node.x + node.width / 2, y: node.y }
  const path = getCurve(from, to, isVertical)
  const labelPath = from.x <= to.x ? path : getCurve(to, from, isVertical)
  return { id: `${source.path}->${node.path}`, path, labelPath, label: one.label, relation: one.relation }
}

/** Where every note of the neighborhood is drawn, and the curve to each from the note it hangs from. */
export function getPlexLayout(hood: Neighborhood): PlexLayout {
  const focus = createNode(hood.focus.path, hood.focus.title, 'focus', FOCUS)
  const placed = placeNeighbors(hood.related)
  const byPath = new Map(placed.map((node) => [node.path, node]))
  const edges = hood.related.flatMap((one) => {
    const node = byPath.get(one.path)
    const source = one.relation === 'sibling' ? byPath.get(one.through) : focus
    return node && source ? [createEdge(one, node, source)] : []
  })
  return { nodes: [focus, ...placed], edges }
}
