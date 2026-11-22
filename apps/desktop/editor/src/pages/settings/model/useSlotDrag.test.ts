import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import type { Clock } from '@numen/ui'
import { moveSlotRow, type SlotRow } from '../lib/slotLayout'
import type { SlotGroup } from '../types'
import { useSlotDrag } from './useSlotDrag'

const group: SlotGroup = { slot: 'titleBar.commands', title: 'Actions', contributors: [] }
const PITCH = 28

const entry = (key: string): SlotRow => ({
  kind: 'entry',
  key,
  label: key,
  pluginName: 'Alpha',
  isHidden: false,
})
const separator = (n: number): SlotRow => ({ kind: 'separator', key: `separator:${n}` })

function createClock() {
  let queue: ((now: number) => void)[] = []
  const clock: Clock = {
    now: () => 0,
    schedule: (run) => queue.push(run),
    cancel: () => {},
  }
  const frame = (): void => {
    const run = queue
    queue = []
    run.forEach((one) => one(0))
  }
  return { clock, frame }
}

function createList(count: number) {
  const list = document.createElement('ul')
  const rows = Array.from({ length: count }, (_, at) => {
    const row = document.createElement('li')
    const grip = document.createElement('button')
    row.append(grip)
    row.getBoundingClientRect = () => new DOMRect(0, at * PITCH, 100, PITCH)
    list.append(row)
    return { row, grip }
  })
  document.body.append(list)
  return { list, rows }
}

function setup(initial: SlotRow[]) {
  const { clock, frame } = createClock()
  let rows: readonly SlotRow[] = initial
  const { list, rows: dom } = createList(initial.length)
  const scope = effectScope()
  const drag = scope.run(() =>
    useSlotDrag(
      {
        getRows: () => rows,
        move: (_group, from, to) => {
          rows = moveSlotRow(rows, from, to)
          return rows
        },
      },
      clock,
    ),
  )!

  const pointer = (type: string, y: number, pointerId = 1): void => {
    window.dispatchEvent(new PointerEvent(type, { clientY: y, pointerId, bubbles: true }))
  }
  const press = (at: number, y: number): void => {
    const grip = dom[at]!.grip
    grip.addEventListener('pointerdown', (event) => drag.onGripPress(group, at, event), {
      once: true,
    })
    grip.dispatchEvent(
      new PointerEvent('pointerdown', { clientY: y, pointerId: 1, button: 0, bubbles: true }),
    )
  }
  const transitionEnd = (key: string): void => {
    const row = dom[0]!.row
    drag.onRowTransitionEnd(
      { target: row, currentTarget: row, propertyName: 'transform' } as unknown as TransitionEvent,
      key,
    )
  }
  const isLifted = (key: string, at: number): boolean =>
    drag.getRowClass(group, key)['plugin-slots__row--lifted'] === true && at >= 0

  return {
    drag,
    dom,
    frame,
    pointer,
    press,
    transitionEnd,
    isLifted,
    getRows: () => rows,
    list,
    scope,
  }
}

const reduceMotion = (isReduced: boolean): void => {
  window.matchMedia = vi.fn(() => ({ matches: isReduced })) as unknown as typeof window.matchMedia
}

afterEach(() => {
  document.body.replaceChildren()
  Reflect.deleteProperty(window, 'matchMedia')
})

describe('the row dropped in a slot', () => {
  it('is let go on the next frame where motion is reduced, since no transition ends', () => {
    reduceMotion(true)
    const page = setup([entry('a'), entry('b'), entry('c')])
    page.press(0, 14)
    page.pointer('pointermove', 30)
    page.pointer('pointermove', 44)
    page.pointer('pointerup', 44)
    page.frame()

    expect(page.isLifted('a', 1)).toBe(false)
    expect(page.drag.getRowStyle(group, 'a', 1)).toBeUndefined()
    page.scope.stop()
  })

  it('is let go on the next frame where it lands exactly where it was dropped', () => {
    reduceMotion(false)
    const page = setup([entry('a'), entry('b'), entry('c')])
    page.press(0, 14)
    page.pointer('pointermove', 30)
    page.pointer('pointermove', 14)
    page.pointer('pointerup', 14)

    expect(page.isLifted('a', 0)).toBe(true)
    page.frame()

    expect(page.isLifted('a', 0)).toBe(false)
    page.scope.stop()
  })

  it('stays lifted until its own transition ends where motion runs', () => {
    reduceMotion(false)
    const page = setup([entry('a'), entry('b'), entry('c')])
    page.press(0, 14)
    page.pointer('pointermove', 30)
    page.pointer('pointermove', 44)
    page.pointer('pointerup', 44)
    page.frame()
    page.frame()

    expect(page.drag.getRowClass(group, 'a')['plugin-slots__row--settling']).toBe(true)
    page.transitionEnd('a')
    expect(page.drag.getRowClass(group, 'a')['plugin-slots__row--settling']).toBe(false)
    page.scope.stop()
  })

  it('is not let go by the transition that ends on another row', () => {
    reduceMotion(false)
    const page = setup([entry('a'), entry('b'), entry('c')])
    page.press(0, 14)
    page.pointer('pointermove', 30)
    page.pointer('pointermove', 44)
    page.pointer('pointerup', 44)
    page.frame()
    page.frame()
    page.transitionEnd('b')

    expect(page.drag.getRowClass(group, 'a')['plugin-slots__row--settling']).toBe(true)
    page.scope.stop()
  })
})

describe('a separator dragged past another', () => {
  it('lands the row that keeps its key at the place the rows give it, not where it was dropped', () => {
    reduceMotion(false)
    const page = setup([separator(0), entry('x'), separator(1)])
    page.press(2, 70)
    page.pointer('pointermove', 50)
    page.pointer('pointermove', 10)
    page.pointer('pointerup', 10)

    expect(page.getRows().map((one) => one.key)).toStrictEqual(['separator:0', 'separator:1', 'x'])
    expect(page.drag.getRowStyle(group, 'separator:1', 1)).toStrictEqual({
      transform: `translate3d(0, ${-PITCH}px, 0)`,
    })
    page.scope.stop()
  })
})

describe('a drag of a row of a slot that is abandoned', () => {
  it('commits nothing when Escape is pressed, and clears the transform of the held row', () => {
    reduceMotion(false)
    const page = setup([entry('a'), entry('b'), entry('c')])
    page.press(0, 14)
    page.pointer('pointermove', 30)
    page.pointer('pointermove', 60)
    expect(page.dom[0]!.row.style.transform).not.toBe('')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    page.pointer('pointerup', 60)

    expect(page.dom[0]!.row.style.transform).toBe('')
    expect(page.getRows().map((one) => one.key)).toStrictEqual(['a', 'b', 'c'])
    page.scope.stop()
  })

  it('commits nothing when the rows of the slot change under the pointer', () => {
    reduceMotion(false)
    const rows: SlotRow[] = [entry('a'), entry('b'), entry('c')]
    const { clock } = createClock()
    const { rows: dom } = createList(3)
    const move = vi.fn((): readonly SlotRow[] => rows)
    const scope = effectScope()
    const drag = scope.run(() => useSlotDrag({ getRows: () => rows, move }, clock))!
    const grip = dom[0]!.grip
    grip.addEventListener('pointerdown', (event) => drag.onGripPress(group, 0, event), {
      once: true,
    })
    grip.dispatchEvent(
      new PointerEvent('pointerdown', { clientY: 14, pointerId: 1, button: 0, bubbles: true }),
    )
    window.dispatchEvent(new PointerEvent('pointermove', { clientY: 30, pointerId: 1 }))
    window.dispatchEvent(new PointerEvent('pointermove', { clientY: 60, pointerId: 1 }))
    expect(dom[0]!.row.style.transform).not.toBe('')

    rows.splice(1, 1)
    window.dispatchEvent(new PointerEvent('pointermove', { clientY: 62, pointerId: 1 }))
    window.dispatchEvent(new PointerEvent('pointerup', { clientY: 62, pointerId: 1 }))

    expect(dom[0]!.row.style.transform).toBe('')
    expect(move).not.toHaveBeenCalled()
    scope.stop()
  })
})
