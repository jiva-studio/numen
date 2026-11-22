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
    cancel: vi.fn(),
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
  const transition = (on: 'Run' | 'End', key: string): void => {
    const row = dom[0]!.row
    const event = { target: row, currentTarget: row, propertyName: 'transform' }
    drag[`onRowTransition${on}`](event as unknown as TransitionEvent, key)
  }
  const isLifted = (key: string, at: number): boolean =>
    drag.getRowClass(group, key)['plugin-slots__row--lifted'] === true && at >= 0

  return {
    drag,
    dom,
    frame,
    pointer,
    press,
    transitionRun: (key: string) => transition('Run', key),
    transitionEnd: (key: string) => transition('End', key),
    isLifted,
    getRows: () => rows,
    list,
    scope,
  }
}

afterEach(() => {
  document.body.replaceChildren()
})

const dropOneRowDown = (page: ReturnType<typeof setup>): void => {
  page.press(0, 14)
  page.pointer('pointermove', 30)
  page.pointer('pointermove', 44)
  page.pointer('pointerup', 44)
}

const isSettling = (page: ReturnType<typeof setup>, key: string): boolean =>
  page.drag.getRowClass(group, key)['plugin-slots__row--settling'] === true

describe('the row dropped in a slot', () => {
  it('clears settling on the third frame where no transition runs', () => {
    const page = setup([entry('a'), entry('b'), entry('c')])
    dropOneRowDown(page)
    page.frame()
    page.frame()
    expect(page.isLifted('a', 1)).toBe(false)
    expect(isSettling(page, 'a')).toBe(true)
    page.frame()

    expect(isSettling(page, 'a')).toBe(false)
    expect(page.drag.getRowStyle(group, 'a', 1)).toBeUndefined()
    page.scope.stop()
  })

  it('clears settling when the offset to the landing is zero', () => {
    const page = setup([entry('a'), entry('b'), entry('c')])
    page.press(0, 14)
    page.pointer('pointermove', 30)
    page.pointer('pointermove', 14)
    page.pointer('pointerup', 14)

    expect(page.isLifted('a', 0)).toBe(true)
    page.frame()
    page.frame()
    page.frame()

    expect(page.isLifted('a', 0)).toBe(false)
    expect(isSettling(page, 'a')).toBe(false)
    page.scope.stop()
  })

  it('stays until its own transition ends where motion runs', () => {
    const page = setup([entry('a'), entry('b'), entry('c')])
    dropOneRowDown(page)
    page.frame()
    page.frame()
    page.transitionRun('a')
    page.frame()

    expect(isSettling(page, 'a')).toBe(true)
    page.transitionEnd('a')
    expect(isSettling(page, 'a')).toBe(false)
    page.scope.stop()
  })

  it('keeps settling when the transition of another row ends', () => {
    const page = setup([entry('a'), entry('b'), entry('c')])
    dropOneRowDown(page)
    page.frame()
    page.frame()
    page.transitionRun('a')
    page.transitionEnd('b')

    expect(isSettling(page, 'a')).toBe(true)
    page.scope.stop()
  })
})

describe('a separator dragged past another', () => {
  it('lands the row that keeps its key at the place the rows give it, not where it was dropped', () => {
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
  it('commits nothing when Escape is pressed', () => {
    const page = setup([entry('a'), entry('b'), entry('c')])
    page.press(0, 14)
    page.pointer('pointermove', 30)
    page.pointer('pointermove', 60)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    page.pointer('pointerup', 60)

    expect(page.getRows().map((one) => one.key)).toStrictEqual(['a', 'b', 'c'])
    page.scope.stop()
  })

  it('commits nothing when the rows of the slot change under the pointer', () => {
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

    rows.splice(1, 1)
    window.dispatchEvent(new PointerEvent('pointermove', { clientY: 62, pointerId: 1 }))
    window.dispatchEvent(new PointerEvent('pointerup', { clientY: 62, pointerId: 1 }))

    expect(move).not.toHaveBeenCalled()
    scope.stop()
  })
})
