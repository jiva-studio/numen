import { effectScope, type EffectScope } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import { stubClock } from '@/shared/fixtures/clock'
import { flatten, type Row } from '../lib/row'
import { useRowDrag, type RowDragEmit, type RowDragState, type TreeMetrics } from './drag'

const ROWS: readonly Row[] = [
  {
    id: 'work',
    name: 'Work',
    hasChildren: true,
    rows: [
      { id: 'plans', name: 'Plans', hasChildren: false },
      { id: 'pages', name: 'Pages', hasChildren: false },
    ],
  },
  { id: 'empty', name: 'Empty', hasChildren: true },
  { id: 'loose', name: 'Loose', hasChildren: false },
]

const METRICS: TreeMetrics = {
  hoverRect: { left: 0, right: 200, top: 100, bottom: 220 },
  top: 100,
  height: 24,
}

let scope: EffectScope | undefined

afterEach(() => {
  scope?.stop()
  scope = undefined
})

const setup = (measure: () => TreeMetrics | null = () => METRICS) => {
  const notifiedCalls: unknown[][] = []
  const emit: RowDragEmit = (...args: unknown[]) => {
    notifiedCalls.push(args)
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
      emit,
    }),
  ) as RowDragState
  return { drag, notified: notifiedCalls }
}

const pointer = (kind: string, x: number, y: number): void => {
  window.dispatchEvent(new MouseEvent(kind, { clientX: x, clientY: y }))
}

const startPress = (drag: RowDragState, rows: readonly string[]): void => {
  drag.startPress(
    rows,
    new MouseEvent('pointerdown', { clientX: 10, clientY: 150 }) as PointerEvent,
  )
}

describe('initial state', () => {
  it('starts with no dragged rows, no drop target, and no label', () => {
    const { drag } = setup()

    expect(drag.into.value).toBeNull()
    expect(drag.before.value).toBeNull()
    expect([...drag.draggedIds.value]).toStrictEqual([])
    expect(drag.label.value).toBeNull()
    expect(drag.hasMoved.value).toBe(false)
  })
})

describe('active drag', () => {
  it('sets the drop target inside a folder row and tracks dragged row ids', () => {
    const { drag } = setup()
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 112)

    expect(drag.at.value).toStrictEqual({ into: 'work' })
    expect(drag.into.value).toBe('work')
    expect(drag.before.value).toBeNull()
    expect([...drag.draggedIds.value]).toStrictEqual(['loose'])
    expect(drag.hasMoved.value).toBe(true)
    expect(drag.label.value?.label).toBe('Loose')
  })

  it('sets the drop target before a row when near its top edge', () => {
    const { drag } = setup()
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 102)

    expect(drag.at.value).toStrictEqual({ before: 'work' })
    expect(drag.before.value).toBe('work')
    expect(drag.into.value).toBeNull()
  })

  it('sets the drop target to root past the last row', () => {
    const { drag } = setup()
    startPress(drag, ['work'])
    pointer('pointermove', 10, 218)

    expect(drag.at.value).toStrictEqual({ into: null })
    expect(drag.into.value).toBeNull()
    expect(drag.before.value).toBeNull()
  })

  it('clears draggedIds on pointerup', () => {
    const { drag } = setup()
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 112)
    pointer('pointerup', 10, 112)

    expect([...drag.draggedIds.value]).toStrictEqual([])
  })

  it('returns a null drop target when measure returns null', () => {
    const { drag, notified } = setup(() => null)
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 112)
    pointer('pointerup', 10, 112)

    expect(drag.at.value).toBeNull()
    expect(notified).toStrictEqual([['drag', ['loose']], ['drop']])
  })

  it('rejects dropping a row into its own descendant', () => {
    const { drag } = setup()
    startPress(drag, ['work'])
    pointer('pointermove', 10, 136)

    expect(drag.at.value).toBeNull()
  })
})

describe('hover bounds', () => {
  const canLandAt = (x: number, y: number): boolean => {
    const { drag } = setup()
    startPress(drag, ['loose'])
    pointer('pointermove', x, y)
    return drag.at.value !== null
  }

  it('accepts points on each edge of hoverRect', () => {
    expect(canLandAt(0, 150)).toBe(true)
    expect(canLandAt(200, 150)).toBe(true)
    expect(canLandAt(10, 100)).toBe(true)
    expect(canLandAt(10, 220)).toBe(true)
  })

  it('rejects points 1px outside any edge of hoverRect', () => {
    expect(canLandAt(-1, 150)).toBe(false)
    expect(canLandAt(201, 150)).toBe(false)
    expect(canLandAt(10, 99)).toBe(false)
    expect(canLandAt(10, 221)).toBe(false)
  })

  it('rejects points horizontally outside hoverRect', () => {
    expect(canLandAt(300, 150)).toBe(false)
    expect(canLandAt(-50, 150)).toBe(false)
  })

  it('rejects points vertically outside hoverRect', () => {
    expect(canLandAt(10, 50)).toBe(false)
    expect(canLandAt(10, 300)).toBe(false)
  })
})

describe('scrolled list top', () => {
  it('rejects points above hoverRect.top when list top is scrolled above it', () => {
    const { drag } = setup(() => ({ ...METRICS, top: 88 }))
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 95)

    expect(drag.at.value).toBeNull()
  })

  it('accepts points at hoverRect.top when list top is scrolled above it', () => {
    const { drag } = setup(() => ({ ...METRICS, top: 88 }))
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 100)

    expect(drag.at.value).toStrictEqual({ into: 'work' })
  })
})

describe('pointerup and pointercancel', () => {
  it('emits move and drop on pointerup over a row', () => {
    const { drag, notified } = setup()
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 112)
    pointer('pointerup', 10, 112)

    expect(notified).toStrictEqual([
      ['drag', ['loose']],
      ['move', ['loose'], { into: 'work' }],
      ['drop'],
    ])
  })

  it('resolves the drop target at the pointerup position', () => {
    const { drag, notified } = setup()
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 112)
    pointer('pointerup', 10, 400)

    expect(notified).toStrictEqual([['drag', ['loose']], ['drop']])
  })

  it('emits drop without move on pointercancel', () => {
    const { drag, notified } = setup()
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 112)
    pointer('pointercancel', 10, 112)

    expect(notified).toStrictEqual([['drag', ['loose']], ['drop']])
  })

  it('emits only drop when pointerup is outside the tree', () => {
    const { drag, notified } = setup()
    startPress(drag, ['loose'])
    pointer('pointermove', 10, 400)
    pointer('pointerup', 10, 400)

    expect(notified).toStrictEqual([['drag', ['loose']], ['drop']])
  })
})
