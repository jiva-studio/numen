/**
 * What a drag over the tree makes of the pointer: where a drop would land, which
 * rows are lifted, and what is told once the rows are let go.
 *
 * The tree stands 200 across and 120 down, 100 from the top of the window, one
 * row every 24: work, plans, notes, empty, loose.
 */
import { effectScope, type EffectScope } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import { stubClock } from '@/shared/fixtures/clock'
import { flatten, type Row } from '../lib/row'
import { useRowDrag, type RowDragState, type RowDragTell, type TreeMetrics } from './drag'

const ROWS: readonly Row[] = [
  {
    id: 'work',
    name: 'Work',
    hasChildren: true,
    rows: [
      { id: 'plans', name: 'Plans', hasChildren: false },
      { id: 'notes', name: 'Notes', hasChildren: false },
    ],
  },
  { id: 'empty', name: 'Empty', hasChildren: true },
  { id: 'loose', name: 'Loose', hasChildren: false },
]

const METRICS: TreeMetrics = {
  over: { left: 0, right: 200, top: 100, bottom: 220 },
  top: 100,
  height: 24,
}

let scope: EffectScope | undefined

afterEach(() => {
  scope?.stop()
  scope = undefined
})

const setup = (measure: () => TreeMetrics | null = () => METRICS) => {
  const told: unknown[][] = []
  const tell: RowDragTell = (...args: unknown[]) => {
    told.push(args)
  }
  const world = stubClock()
  scope = effectScope()
  const drag = scope.run(() =>
    useRowDrag({
      getRows: () => ROWS,
      getShownRows: () => flatten(ROWS, new Set(['work'])),
      measure,
      getThreshold: () => 4,
      getClock: () => world.clock,
      getCountWords: () => (rows) => `${rows} rows`,
      tell,
    }),
  ) as RowDragState
  return { drag, told }
}

const pointer = (kind: string, x: number, y: number): void => {
  window.dispatchEvent(new MouseEvent(kind, { clientX: x, clientY: y }))
}

/** A press on a row at the middle of the tree. */
const lift = (drag: RowDragState, rows: readonly string[]): void => {
  drag.lift(rows, new MouseEvent('pointerdown', { clientX: 10, clientY: 150 }) as PointerEvent)
}

describe('before a press', () => {
  it('holds no rows, lands nowhere and shows nothing', () => {
    const { drag } = setup()

    expect(drag.into.value).toBeNull()
    expect(drag.before.value).toBeNull()
    expect([...drag.lifted.value]).toStrictEqual([])
    expect(drag.label.value).toBeNull()
    expect(drag.hasMoved.value).toBe(false)
  })
})

describe('a press that has become a drag', () => {
  it('lands inside the row it is over, and holds the rows lifted', () => {
    const { drag } = setup()
    lift(drag, ['loose'])
    pointer('pointermove', 10, 112)

    expect(drag.at.value).toStrictEqual({ into: 'work' })
    expect(drag.into.value).toBe('work')
    expect(drag.before.value).toBeNull()
    expect([...drag.lifted.value]).toStrictEqual(['loose'])
    expect(drag.hasMoved.value).toBe(true)
    expect(drag.label.value?.label).toBe('Loose')
  })

  it('lands above the row it is over the top of', () => {
    const { drag } = setup()
    lift(drag, ['loose'])
    pointer('pointermove', 10, 102)

    expect(drag.at.value).toStrictEqual({ before: 'work' })
    expect(drag.before.value).toBe('work')
    expect(drag.into.value).toBeNull()
  })

  it('lands at the top level past the last row', () => {
    const { drag } = setup()
    lift(drag, ['work'])
    pointer('pointermove', 10, 218)

    expect(drag.at.value).toStrictEqual({ into: null })
    expect(drag.into.value).toBeNull()
    expect(drag.before.value).toBeNull()
  })

  it('holds the rows lifted only while the pointer is down', () => {
    const { drag } = setup()
    lift(drag, ['loose'])
    pointer('pointermove', 10, 112)
    pointer('pointerup', 10, 112)

    expect([...drag.lifted.value]).toStrictEqual([])
  })

  it('lands nowhere where the tree is not drawn', () => {
    const { drag, told } = setup(() => null)
    lift(drag, ['loose'])
    pointer('pointermove', 10, 112)
    pointer('pointerup', 10, 112)

    expect(drag.at.value).toBeNull()
    expect(told).toStrictEqual([['drag', ['loose']], ['drop']])
  })

  it('lands nowhere that would put a row inside itself', () => {
    const { drag } = setup()
    lift(drag, ['work'])
    pointer('pointermove', 10, 136)

    expect(drag.at.value).toBeNull()
  })
})

describe('how far the tree reaches', () => {
  const canLandAt = (x: number, y: number): boolean => {
    const { drag } = setup()
    lift(drag, ['loose'])
    pointer('pointermove', x, y)
    return drag.at.value !== null
  }

  it('takes in each edge', () => {
    expect(canLandAt(0, 150)).toBe(true)
    expect(canLandAt(200, 150)).toBe(true)
    expect(canLandAt(10, 100)).toBe(true)
    expect(canLandAt(10, 220)).toBe(true)
  })

  it('takes in nothing beyond any edge', () => {
    expect(canLandAt(-1, 150)).toBe(false)
    expect(canLandAt(201, 150)).toBe(false)
    expect(canLandAt(10, 99)).toBe(false)
    expect(canLandAt(10, 221)).toBe(false)
  })

  it('takes in nothing alongside it, level with its rows', () => {
    expect(canLandAt(300, 150)).toBe(false)
    expect(canLandAt(-50, 150)).toBe(false)
  })

  it('takes in nothing above or below it, in line with its rows', () => {
    expect(canLandAt(10, 50)).toBe(false)
    expect(canLandAt(10, 300)).toBe(false)
  })
})

describe('a tree whose rows begin below its top edge', () => {
  it('takes in nothing above the edge, though a row remains there', () => {
    const { drag } = setup(() => ({ ...METRICS, top: 88 }))
    lift(drag, ['loose'])
    pointer('pointermove', 10, 95)

    expect(drag.at.value).toBeNull()
  })

  it('takes in the edge itself', () => {
    const { drag } = setup(() => ({ ...METRICS, top: 88 }))
    lift(drag, ['loose'])
    pointer('pointermove', 10, 100)

    expect(drag.at.value).toStrictEqual({ into: 'work' })
  })
})

describe('letting go', () => {
  it('notifies where the rows landed, then that they were let go', () => {
    const { drag, told } = setup()
    lift(drag, ['loose'])
    pointer('pointermove', 10, 112)
    pointer('pointerup', 10, 112)

    expect(told).toStrictEqual([
      ['drag', ['loose']],
      ['move', ['loose'], { into: 'work' }],
      ['drop'],
    ])
  })

  it('notifies only that they were let go where they landed nowhere', () => {
    const { drag, told } = setup()
    lift(drag, ['loose'])
    pointer('pointermove', 10, 400)
    pointer('pointerup', 10, 400)

    expect(told).toStrictEqual([['drag', ['loose']], ['drop']])
  })
})
