/** Where the nodes go: parents above, children below, jumps left, siblings right. */
import { describe, expect, it } from 'vitest'
import { rowsAndColumns, type Seating, type Widths } from './placement'
import { DEFAULT_OPTIONS, type PlexOptions } from './options'
import type { Limits, RoleLimits } from './limits'
import type { PlacedNode, PlexNode } from '../node'
import { RELATED_SEATS, type PlexRelatedSeat } from '../seat'

const { nodeSize, focusSize, gap, lineGap, focusGap } = DEFAULT_OPTIONS

const FOCUS: PlacedNode = {
  id: 'focus',
  title: 'The focus',
  seat: 'focus',
  x: 0,
  y: 0,
  width: focusSize.width,
  height: focusSize.height,
  order: 0,
  opacity: 1,
}

/** Nodes of one seat, named after it. */
const createNodes = (seat: PlexRelatedSeat, count: number): readonly PlexNode[] =>
  Array.from({ length: count }, (_, at) => ({ id: `${seat}-${at}`, title: `${seat} ${at}`, seat }))

const limits = (each: RoleLimits): Limits =>
  Object.fromEntries(RELATED_SEATS.map((seat) => [seat, each])) as Limits

/** Every box the width its seat is drawn at. */
const evenly: Widths = () => nodeSize.width

const place = (
  seats: Seating,
  perLine = DEFAULT_OPTIONS.maxPerLine,
  options: PlexOptions = DEFAULT_OPTIONS,
): PlacedNode[] =>
  rowsAndColumns.place(
    seats,
    FOCUS,
    options,
    limits({ perLine, lines: DEFAULT_OPTIONS.maxLines }),
    evenly,
  )

const of = (nodes: readonly PlacedNode[], seat: PlexRelatedSeat) =>
  nodes.filter((node) => node.seat === seat)

describe('what comes back', () => {
  it('is every node it was given, and never the focus', () => {
    const placed = place({ child: createNodes('child', 3), parent: createNodes('parent', 2) })

    expect(placed).toHaveLength(5)
    expect(placed.some((node) => node.id === FOCUS.id)).toBe(false)
  })

  it('hands each node back with the identity and title it came in with', () => {
    const [one] = place({ child: createNodes('child', 1) })

    expect(one?.id).toBe('child-0')
    expect(one?.title).toBe('child 0')
    expect(one?.seat).toBe('child')
  })

  it('draws every box the width it was told and the height of a node', () => {
    const placed = place({ child: createNodes('child', 3) })

    expect(placed.map((node) => node.width)).toStrictEqual([144, 144, 144])
    expect(placed.map((node) => node.height)).toStrictEqual([36, 36, 36])
  })

  it('places nothing for a seating holding nothing', () => {
    expect(place({})).toStrictEqual([])
  })
})

describe('a row', () => {
  it('runs below the focus for children and above it for parents', () => {
    const placed = place({ child: createNodes('child', 2), parent: createNodes('parent', 2) })

    expect(of(placed, 'child').every((node) => node.y > 0)).toBe(true)
    expect(of(placed, 'parent').every((node) => node.y < 0)).toBe(true)
  })

  it('is centred on the focus', () => {
    const row = of(place({ child: createNodes('child', 3) }), 'child')
    const left = row[0]!.x - row[0]!.width / 2
    const right = row[2]!.x + row[2]!.width / 2

    expect(left).toBeCloseTo(-right)
  })

  it('stands one gap between neighbours', () => {
    const row = of(place({ child: createNodes('child', 3) }), 'child')

    expect(row[1]!.x - row[0]!.x).toBeCloseTo(nodeSize.width + gap)
    expect(row[2]!.x - row[1]!.x).toBeCloseTo(nodeSize.width + gap)
  })

  it('clears the focus by the gap kept for it', () => {
    const [one] = of(place({ child: createNodes('child', 1) }), 'child')

    expect(one!.y - nodeSize.height / 2).toBeCloseTo(focusSize.height / 2 + focusGap)
  })

  it('numbers its nodes outward from the first', () => {
    const row = of(place({ child: createNodes('child', 3) }), 'child')

    expect(row.map((node) => node.order)).toStrictEqual([0, 1, 2])
  })
})

describe('a row too long for one line', () => {
  it('wraps onto a second, each line standing one box beyond the last', () => {
    const row = of(place({ child: createNodes('child', 4) }, 2), 'child')
    const lines = [...new Set(row.map((node) => node.y))].sort((a, b) => a - b)

    expect(lines).toHaveLength(2)
    expect(lines[1]! - lines[0]!).toBeCloseTo(nodeSize.height + lineGap)
  })

  it('shares its nodes out evenly, every line within one of every other', () => {
    const row = of(place({ child: createNodes('child', 5) }, 3), 'child')
    const counts = [
      ...new Map<number, number>(
        row.map((node) => [node.y, row.filter((each) => each.y === node.y).length]),
      ).values(),
    ]

    expect(Math.max(...counts) - Math.min(...counts)).toBeLessThanOrEqual(1)
  })

  it('keeps numbering across the lines', () => {
    const row = of(place({ child: createNodes('child', 4) }, 2), 'child')

    expect(row.map((node) => node.order)).toStrictEqual([0, 1, 2, 3])
  })
})

describe('a column', () => {
  it('runs left of the focus for jumps and right of it for siblings', () => {
    const placed = place({ jump: createNodes('jump', 2), sibling: createNodes('sibling', 2) })

    expect(of(placed, 'jump').every((node) => node.x < 0)).toBe(true)
    expect(of(placed, 'sibling').every((node) => node.x > 0)).toBe(true)
  })

  it('is centred on the focus, one gap between neighbours', () => {
    const column = of(place({ sibling: createNodes('sibling', 3) }), 'sibling')

    expect(column[1]!.y).toBeCloseTo(0)
    expect(column[1]!.y - column[0]!.y).toBeCloseTo(nodeSize.height + gap)
    expect(column[2]!.y - column[1]!.y).toBeCloseTo(nodeSize.height + gap)
  })

  it('turns the same edge towards the focus, whatever a box measures', () => {
    const widths: Widths = (node) => (node.id.endsWith('1') ? 240 : nodeSize.width)
    const column = rowsAndColumns
      .place(
        { sibling: createNodes('sibling', 2) },
        FOCUS,
        DEFAULT_OPTIONS,
        limits({ perLine: 5, lines: 4 }),
        widths,
      )
      .filter((node) => node.seat === 'sibling')

    const edges = column.map((node) => node.x - node.width / 2)
    expect(edges[0]).toBeCloseTo(edges[1]!)
  })

  it('stands clear of a row it reaches alongside', () => {
    const alone = of(place({ sibling: createNodes('sibling', 5) }), 'sibling')
    const beside = of(
      place({ sibling: createNodes('sibling', 5), child: createNodes('child', 5) }),
      'sibling',
    )

    expect(beside[0]!.x).toBeGreaterThan(alone[0]!.x)
  })

  it('keeps its own distance from a row it never reaches alongside', () => {
    const alone = of(place({ sibling: createNodes('sibling', 2) }), 'sibling')
    const beside = of(
      place({ sibling: createNodes('sibling', 2), child: createNodes('child', 5) }),
      'sibling',
    )

    expect(beside[0]!.x).toBeCloseTo(alone[0]!.x)
  })

  it('gives both sides the one clearance, however tall each is', () => {
    const placed = place({ jump: createNodes('jump', 1), sibling: createNodes('sibling', 4) })
    const jump = of(placed, 'jump')[0]!
    const sibling = of(placed, 'sibling')[0]!

    expect(Math.abs(jump.x) - jump.width / 2).toBeCloseTo(Math.abs(sibling.x) - sibling.width / 2)
  })

  it('wraps into a second line clear of the widest box of the first', () => {
    const column = of(place({ sibling: createNodes('sibling', 4) }, 2), 'sibling')
    const lines = [...new Set(column.map((node) => node.x))].sort((a, b) => a - b)

    expect(lines).toHaveLength(2)
    expect(lines[1]! - lines[0]!).toBeCloseTo(nodeSize.width + lineGap)
  })
})

describe('rows and columns together', () => {
  it('never lays a column over a row', () => {
    const placed = place({ child: createNodes('child', 5), sibling: createNodes('sibling', 3) })
    const isOverlapping = (one: PlacedNode, two: PlacedNode) =>
      Math.abs(one.x - two.x) < (one.width + two.width) / 2 &&
      Math.abs(one.y - two.y) < (one.height + two.height) / 2

    for (const row of of(placed, 'child')) {
      for (const column of of(placed, 'sibling')) {
        expect(isOverlapping(row, column)).toBe(false)
      }
    }
  })

  it('leaves every box fully there', () => {
    const placed = place({
      child: createNodes('child', 4),
      parent: createNodes('parent', 2),
      jump: createNodes('jump', 2),
      sibling: createNodes('sibling', 2),
    })

    expect(placed.every((node) => node.opacity === 1)).toBe(true)
  })
})
