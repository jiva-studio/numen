/** The arrangement, tested without rendering it. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { neighbourhoods } from '../fixtures/neighbourhoods'
import type {
  PlacedNode,
  PlexFrame,
  PlexNeighbourhood,
  PlexNode,
  PlexRelatedSeat,
  PlexSeat,
} from '../model'
import type { Placement } from './placement'

const every = Object.entries(neighbourhoods)

/**
 * A width that follows the title and needs no font. Nothing in the arrangement
 * is allowed to assume one size for every box, and the fixtures carry titles
 * from one letter to a sentence.
 */
const byTitle = (node: PlexNode): number => 40 + node.title.length * 7

/** A neighbourhood whose titles are picked for their lengths alone. */
const titled = (seat: PlexRelatedSeat, ...titles: string[]): PlexNeighbourhood => ({
  nodes: [
    { id: 'focus', title: 'Focus', seat: 'focus' },
    ...titles.map((title, index) => ({ id: `${seat}-${index}`, title, seat })),
  ],
  edges: [],
})

/** The nodes of one seat, cut into the lines they were laid out on. */
const inLines = (nodes: readonly PlacedNode[], perLine: number): PlacedNode[][] => {
  const lines: PlacedNode[][] = []
  for (const node of nodes) {
    const line = Math.floor(node.order / perLine)
    ;(lines[line] ??= []).push(node)
  }
  return lines
}

const withSeat = (layout: { nodes: readonly PlacedNode[] }, seat: PlexSeat) =>
  layout.nodes.filter((node) => node.seat === seat)

const focusOf = (layout: { nodes: readonly PlacedNode[] }) => {
  const focus = layout.nodes.find((node) => node.seat === 'focus')
  if (!focus) throw new Error('unreachable: every layout places a focus')
  return focus
}

/** Boxes are centred on their coordinates; touching edges is not overlapping. */
function overlaps(a: PlacedNode, b: PlacedNode): boolean {
  return (
    Math.abs(a.x - b.x) < (a.width + b.width) / 2 - 1e-9 &&
    Math.abs(a.y - b.y) < (a.height + b.height) / 2 - 1e-9
  )
}

describe('the contract on the neighbourhood', () => {
  it('refuses a neighbourhood with no focus', () => {
    const nodes = [{ id: 'a', title: 'A', seat: 'child' as const }]
    expect(() => arrangePlex({ nodes, edges: [] })).toThrow(/needs a node with the focus/)
  })

  it('refuses a neighbourhood with two nodes in focus', () => {
    const nodes = [
      { id: 'a', title: 'A', seat: 'focus' as const },
      { id: 'b', title: 'B', seat: 'focus' as const },
    ]
    expect(() => arrangePlex({ nodes, edges: [] })).toThrow(/one focus, not 2/)
  })

  it('refuses the same identifier in two seats', () => {
    const nodes = [
      { id: 'a', title: 'A', seat: 'focus' as const },
      { id: 'b', title: 'B', seat: 'parent' as const },
      { id: 'b', title: 'B again', seat: 'child' as const },
    ]
    expect(() => arrangePlex({ nodes, edges: [] })).toThrow(/appears more than once: b/)
  })
})

describe('where the seats sit', () => {
  const layout = arrangePlex(neighbourhoods.typical)
  const focus = focusOf(layout)

  it('puts the focus at the origin', () => {
    expect(focus.x).toBe(0)
    expect(focus.y).toBe(0)
  })

  it('puts every parent clear above the focus', () => {
    const parents = withSeat(layout, 'parent')
    expect(parents.length).toBeGreaterThan(0)
    for (const parent of parents) {
      expect(parent.y + parent.height / 2).toBeLessThan(-focus.height / 2)
    }
  })

  it('puts every child clear below the focus', () => {
    const children = withSeat(layout, 'child')
    expect(children.length).toBeGreaterThan(0)
    for (const child of children) {
      expect(child.y - child.height / 2).toBeGreaterThan(focus.height / 2)
    }
  })

  it('puts every jump clear to the left and every sibling clear to the right', () => {
    for (const jump of withSeat(layout, 'jump')) {
      expect(jump.x + jump.width / 2).toBeLessThan(-focus.width / 2)
    }
    for (const sibling of withSeat(layout, 'sibling')) {
      expect(sibling.x - sibling.width / 2).toBeGreaterThan(focus.width / 2)
    }
    expect(withSeat(layout, 'jump').length).toBeGreaterThan(0)
    expect(withSeat(layout, 'sibling').length).toBeGreaterThan(0)
  })

  it('clears the rows a column runs alongside, and only those', () => {
    // A column at a fixed distance lands on a row once the row outgrows it.
    const wide = arrangePlex(neighbourhoods.overcrowded)
    const columns = withSeat(wide, 'jump')
    const columnReach = Math.max(...columns.map((n) => Math.abs(n.y) + n.height / 2))
    const alongside = [...withSeat(wide, 'parent'), ...withSeat(wide, 'child')].filter(
      (n) => Math.abs(n.y) - n.height / 2 < columnReach,
    )
    expect(alongside.length).toBeGreaterThan(0)
    const rowsReach = Math.max(...alongside.map((n) => Math.abs(n.x) + n.width / 2))
    for (const column of columns) {
      expect(Math.abs(column.x) - column.width / 2).toBeGreaterThanOrEqual(rowsReach)
    }

    // And a short column is not dragged out to a row it never comes near.
    const short = Math.min(...withSeat(layout, 'jump').map((n) => Math.abs(n.x)))
    const childrenReach = Math.max(
      ...withSeat(layout, 'child').map((n) => Math.abs(n.x) + n.width / 2),
    )
    expect(short).toBeLessThan(childrenReach)
  })
})

describe('a box is as wide as its title needs', () => {
  const { minWidth, nodeSize, focusSize, gap, lineGap, maxPerLine } = DEFAULT_OPTIONS

  it('draws every box at its widest when nothing was measured', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    expect(focusOf(layout).width).toBe(focusSize.width)
    for (const node of withSeat(layout, 'child')) expect(node.width).toBe(nodeSize.width)
  })

  it('gives each box the width it was measured at', () => {
    const measurable = titled('child', 'Domain', 'Use case', 'Aggregate', 'Repository')
    const layout = arrangePlex(measurable, { measure: byTitle })
    for (const node of layout.nodes) expect(node.width).toBe(byTitle(node))
  })

  it('sizes the focus to its title as it sizes any other box', () => {
    const layout = arrangePlex(titled('child', 'Port'), { measure: byTitle })
    const focus = focusOf(layout)
    expect(focus.width).toBe(byTitle(focus))
    expect(focus.width).toBeLessThan(focusSize.width)
  })

  it('clamps a title past the maximum, which each seat has its own of', () => {
    const layout = arrangePlex(neighbourhoods.typical, { measure: () => 4000 })
    expect(focusOf(layout).width).toBe(focusSize.width)
    for (const node of withSeat(layout, 'child')) expect(node.width).toBe(nodeSize.width)
  })

  it('gives a title that measures next to nothing the minimum', () => {
    const layout = arrangePlex(titled('child', '', 'A'), { measure: () => 0 })
    expect(focusOf(layout).width).toBe(minWidth)
    for (const node of withSeat(layout, 'child')) expect(node.width).toBe(minWidth)
  })

  it('leaves a row packed to one gap between boxes, and centred on the focus', () => {
    const mixed = titled('child', 'A', 'Domain', 'Composition root', 'Port', 'Use case')
    const children = withSeat(arrangePlex(mixed, { measure: byTitle }), 'child')

    expect(new Set(children.map((node) => node.width)).size).toBeGreaterThan(1)
    for (let i = 1; i < children.length; i++) {
      const before = children[i - 1]!
      const node = children[i]!
      expect(node.x - node.width / 2 - (before.x + before.width / 2)).toBeCloseTo(gap)
    }

    const left = Math.min(...children.map((node) => node.x - node.width / 2))
    const right = Math.max(...children.map((node) => node.x + node.width / 2))
    expect(left + right).toBeCloseTo(0)
  })

  it('turns one edge of a column towards the focus, whatever the widths', () => {
    const mixed = titled('jump', 'A', 'Dependency inversion', 'Port')
    const jumps = withSeat(arrangePlex(mixed, { measure: byTitle }), 'jump')

    expect(new Set(jumps.map((node) => node.width)).size).toBeGreaterThan(1)
    const near = jumps.map((node) => Math.abs(node.x) - node.width / 2)
    for (const edge of near) expect(edge).toBeCloseTo(near[0]!)
  })

  it('starts each further column beyond the widest of the one before it', () => {
    const jumps = withSeat(
      arrangePlex(neighbourhoods.overcrowded, { measure: byTitle }),
      'jump',
    )
    const lines = inLines(jumps, maxPerLine)
    expect(lines.length).toBeGreaterThan(1)

    const near = (line: PlacedNode[]) => Math.abs(line[0]!.x) - line[0]!.width / 2
    for (let i = 1; i < lines.length; i++) {
      const before = lines[i - 1]!
      const widest = Math.max(...before.map((node) => node.width))
      expect(near(lines[i]!)).toBeCloseTo(near(before) + widest + lineGap)
    }
  })

  it('shows as much as it would have shown unmeasured', () => {
    // How much is admitted is settled from the widest a box gets, so what a
    // title measures never changes the count or what the overflow reports.
    const measured = arrangePlex(neighbourhoods.overcrowded, { measure: byTitle })
    const plain = arrangePlex(neighbourhoods.overcrowded)

    expect(measured.overflow).toStrictEqual(plain.overflow)
    expect(measured.nodes.map((node) => node.id)).toStrictEqual(
      plain.nodes.map((node) => node.id),
    )
  })
})

describe('a label is as long as its words', () => {
  /** One line, running sideways, with words written along it. */
  const labelled: PlexNeighbourhood = {
    nodes: [
      { id: 'focus', title: 'Here', seat: 'focus' },
      { id: 'aside', title: 'Aside', seat: 'jump' },
    ],
    edges: [{ from: 'aside', to: 'focus', label: 'the scene in the assembly' }],
  }

  const wordsOf = (measureLabel?: (label: string) => number) =>
    arrangePlex(labelled, { measureLabel }).edges[0]!.words

  it('asks the measurer for the words a line carries, and for no others', () => {
    const asked: string[] = []
    arrangePlex(labelled, {
      measureLabel: (label) => {
        asked.push(label)
        return 0
      },
    })

    // Once to cut the words to the line, and once to find them room along it.
    expect(asked).toStrictEqual([
      'the scene in the assembly',
      'the scene in the assembly',
    ])
  })

  it('cuts words too long for their line', () => {
    expect(wordsOf((label) => 6 * [...label].length)).toBe('the scen…')
  })

  it('writes no words at all where the line holds none of them', () => {
    expect(wordsOf(() => 4000)).toBeUndefined()
  })

  it('leaves words that fit as they were written', () => {
    expect(wordsOf(() => 1)).toBe('the scene in the assembly')
  })

  it('draws the whole label where nothing measured it', () => {
    expect(wordsOf()).toBe('the scene in the assembly')
  })
})

describe('no two nodes are drawn on top of each other', () => {
  const noneOverlap = (nodes: readonly PlacedNode[]) => {
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        const a = nodes[i]
        const b = nodes[j]
        if (!a || !b) continue
        expect(overlaps(a, b), `${a.id} overlaps ${b.id}`).toBe(false)
      }
    }
  }

  it.each(every)('%s', (_name, neighbourhood: PlexNeighbourhood) => {
    noneOverlap(arrangePlex(neighbourhood).nodes)
  })

  it.each(every)('%s, sized to its titles', (_name, neighbourhood: PlexNeighbourhood) => {
    noneOverlap(arrangePlex(neighbourhood, { measure: byTitle }).nodes)
  })
})

describe('the same input gives the same numbers', () => {
  it.each(every)('%s', (_name, neighbourhood: PlexNeighbourhood) => {
    expect(arrangePlex(neighbourhood)).toStrictEqual(arrangePlex(neighbourhood))
  })

  it.each(every)('%s, sized to its titles', (_name, neighbourhood: PlexNeighbourhood) => {
    expect(arrangePlex(neighbourhood, { measure: byTitle })).toStrictEqual(
      arrangePlex(neighbourhood, { measure: byTitle }),
    )
  })

  it('does not depend on the order the seats arrive in', () => {
    const shuffled = {
      ...neighbourhoods.typical,
      nodes: [...neighbourhoods.typical.nodes].reverse(),
    }
    const bySeat = (n: PlexNeighbourhood) =>
      arrangePlex(n).nodes.filter((node) => node.seat === 'child').map((node) => node.id)
    // Order follows the input rather than a sort; the set of seats is equal.
    expect(bySeat(shuffled)).toHaveLength(bySeat(neighbourhoods.typical).length)
    expect([...bySeat(shuffled)].sort()).toStrictEqual([...bySeat(neighbourhoods.typical)].sort())
  })
})

describe('the lines of a seat are shared out evenly', () => {
  it('keeps every line within one node of every other', () => {
    const children = withSeat(
      arrangePlex(neighbourhoods.typical, { options: { maxPerLine: 4 } }),
      'child',
    )
    const lines = new Map<number, number>()
    for (const node of children) lines.set(node.y, (lines.get(node.y) ?? 0) + 1)

    const lengths = [...lines.values()]
    expect(Math.max(...lengths) - Math.min(...lengths)).toBeLessThanOrEqual(1)
  })

  it('never puts more on a line than the limit allows', () => {
    for (const maxPerLine of [1, 2, 3, 4, 5]) {
      const children = withSeat(
        arrangePlex(neighbourhoods.crowded, { options: { maxPerLine } }),
        'child',
      )
      const lines = new Map<number, number>()
      for (const node of children) lines.set(node.y, (lines.get(node.y) ?? 0) + 1)
      expect(Math.max(...lines.values())).toBeLessThanOrEqual(maxPerLine)
    }
  })
})

describe('order along a line follows the order given', () => {
  it('numbers each seat outward from the focus', () => {
    const children = withSeat(arrangePlex(neighbourhoods.crowded), 'child')
    expect(children.map((node) => node.order)).toStrictEqual(
      children.map((_, index) => index),
    )
  })

  it('lays a row out left to right', () => {
    const children = withSeat(arrangePlex(neighbourhoods.typical), 'child')
    const firstLine = children.filter((node) => node.y === children[0]!.y)
    expect(firstLine.length).toBeGreaterThan(1)
    for (let i = 1; i < firstLine.length; i++) {
      const previous = firstLine[i - 1]
      const current = firstLine[i]
      if (!previous || !current) continue
      expect(current.x).toBeGreaterThan(previous.x)
    }
  })

  it('puts the second line further from the focus than the first', () => {
    const children = withSeat(arrangePlex(neighbourhoods.crowded), 'child')
    const first = children[0]
    const onSecondLine = children.find((node) => node.y !== first!.y)
    expect(first).toBeDefined()
    expect(onSecondLine).toBeDefined()
    expect(onSecondLine!.y).toBeGreaterThan(first!.y)
  })

  it('keeps the relative order when one more node arrives', () => {
    const before = neighbourhoods.typical
    const after: PlexNeighbourhood = {
      nodes: [...before.nodes, { id: 'child-6', title: 'One more', seat: 'child' }],
      edges: [...before.edges, { from: 'focus', to: 'child-6' }],
    }
    const ids = (n: PlexNeighbourhood) =>
      withSeat(arrangePlex(n), 'child').map((node) => node.id)
    expect(ids(after).slice(0, ids(before).length)).toStrictEqual(ids(before))
  })
})

describe('what does not fit is reported rather than dropped silently', () => {
  it('counts the nodes past the last line, per seat', () => {
    const { maxPerLine, maxLines } = DEFAULT_OPTIONS
    const capacity = maxPerLine * maxLines
    const { overflow, nodes } = arrangePlex(neighbourhoods.overcrowded)

    expect(overflow.child).toBe(200 - capacity)
    expect(overflow.jump).toBe(40 - capacity)
    expect(overflow.parent).toBeUndefined() // nine parents fit in four lines of five
    expect(nodes.filter((node) => node.seat === 'child')).toHaveLength(capacity)
  })

  it('reports nothing when everything fits', () => {
    expect(arrangePlex(neighbourhoods.typical).overflow).toStrictEqual({})
  })

  it('drops the edges that lead to a node which did not fit', () => {
    const { nodes, edges } = arrangePlex(neighbourhoods.overcrowded)
    const placed = new Set(nodes.map((node) => node.id))
    for (const edge of edges) {
      expect(placed.has(edge.from)).toBe(true)
      expect(placed.has(edge.to)).toBe(true)
    }
    expect(edges.length).toBeLessThan(neighbourhoods.overcrowded.edges.length)
  })
})

describe('the extent', () => {
  const holdsEveryNode = ({ nodes, extent }: PlexFrame) => {
    for (const node of nodes) {
      expect(node.x - node.width / 2).toBeGreaterThanOrEqual(extent.minX)
      expect(node.x + node.width / 2).toBeLessThanOrEqual(extent.maxX)
      expect(node.y - node.height / 2).toBeGreaterThanOrEqual(extent.minY)
      expect(node.y + node.height / 2).toBeLessThanOrEqual(extent.maxY)
    }
  }

  it.each(every)('contains every node of %s', (_name, neighbourhood: PlexNeighbourhood) => {
    holdsEveryNode(arrangePlex(neighbourhood))
  })

  it.each(every)(
    'contains every node of %s, sized to its titles',
    (_name, neighbourhood: PlexNeighbourhood) => {
      holdsEveryNode(arrangePlex(neighbourhood, { measure: byTitle }))
    },
  )

  it('is the focus box alone when the note stands alone', () => {
    const { extent } = arrangePlex(neighbourhoods.solitary)
    const { width, height } = DEFAULT_OPTIONS.focusSize
    expect(extent).toStrictEqual({
      minX: -width / 2,
      minY: -height / 2,
      maxX: width / 2,
      maxY: height / 2,
    })
  })
})

describe('the options are honoured', () => {
  it('lets the caller change how many fit on a line', () => {
    const narrow = arrangePlex(neighbourhoods.crowded, { options: { maxPerLine: 3 } })
    const children = withSeat(narrow, 'child')
    const onFirstLine = children.filter((node) => node.y === children[0]!.y)
    expect(onFirstLine).toHaveLength(3)
  })

  it('lets the caller send children up and parents down', () => {
    const upside = arrangePlex(neighbourhoods.typical, {
      options: {
        direction: { parent: 'down', child: 'up', jump: 'left', sibling: 'right' },
      },
    })
    for (const child of withSeat(upside, 'child')) expect(child.y).toBeLessThan(0)
    for (const parent of withSeat(upside, 'parent')) expect(parent.y).toBeGreaterThan(0)
  })

  it('takes a curve setting without being told the size of a box', () => {
    // Tuning a curve should not need to know the size of a box.
    const straighter = arrangePlex(neighbourhoods.typical, {
      options: { routing: { curvature: 0 } },
    })
    const edge = straighter.edges[0]!
    expect(edge.control1.y - edge.fromPoint.y).toBe(DEFAULT_OPTIONS.routing.minReach)
  })
})

describe('the arrangement is a strategy, not a shape baked in', () => {
  it('uses rows and columns unless told otherwise', () => {
    const frame = arrangePlex(neighbourhoods.typical)
    expect(withSeat(frame, 'parent').every((n) => n.y < 0)).toBe(true)
  })

  it('lets another strategy place the nodes, and routes its edges anyway', () => {
    // The point is not the ring: a strategy sets coordinates and inherits
    // the limits, the routing and the extent.
    const ring: Placement = {
      name: 'ring',
      place(seating) {
        const all = Object.values(seating).flat()
        return all.map((node, index) => {
          const angle = (index / all.length) * Math.PI * 2
          return {
            ...node,
            x: Math.round(Math.cos(angle) * 300),
            y: Math.round(Math.sin(angle) * 300),
            width: 144,
            height: 36,
            order: index,
            opacity: 1,
          }
        })
      },
    }

    const frame = arrangePlex(neighbourhoods.typical, { placement: ring })
    const focus = focusOf(frame)

    expect(focus.x).toBe(0)
    expect(focus.y).toBe(0)
    expect(frame.nodes).toHaveLength(neighbourhoods.typical.nodes.length)
    // Edges still meet their boxes, without the strategy knowing they exist.
    expect(frame.edges).toHaveLength(neighbourhoods.typical.edges.length)
    for (const edge of frame.edges) {
      expect(Number.isFinite(edge.control1.x)).toBe(true)
    }
  })

  it('does not let a strategy decide how much is shown', () => {
    // Admission is settled before a strategy runs, so the count is the same
    // whichever arrangement drew it.
    const counting: Placement = {
      name: 'counting',
      place: (seating) => Object.values(seating).flat().map((node, index) => ({
        ...node,
        x: index * 200,
        y: 100,
        width: 144,
        height: 36,
        order: index,
        opacity: 1,
      })),
    }

    const { maxPerLine, maxLines } = DEFAULT_OPTIONS
    const frame = arrangePlex(neighbourhoods.overcrowded, { placement: counting })
    expect(frame.overflow.child).toBe(200 - maxPerLine * maxLines)
  })
})
