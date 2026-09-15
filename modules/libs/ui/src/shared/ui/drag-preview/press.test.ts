/**
 * A press followed by hand. jsdom has no pointer, so the events are made here
 * and the frame is a function the test calls itself.
 */
import { effectScope } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { usePressDrag } from './press'
import type { Position } from '@/shared/lib/geometry'
import type { Clock } from '@/shared/lib/clock'

const pointer = (type: string, x: number, y: number) =>
  new PointerEvent(type, { clientX: x, clientY: y, bubbles: true })

/** A press under way, with the frame held until the test lets it come. */
function createPress(threshold = 4) {
  const settle = vi.fn<(item: string, at: Position | null) => void>()
  const begin = vi.fn<(item: string) => void>()
  let next: ((now: number) => void) | null = null

  /** A clock whose next frame comes when the test says so. */
  const clock: Clock = {
    now: () => 0,
    schedule: (run) => {
      next = run
      return 0
    },
    cancel: () => {
      next = null
    },
  }

  const scope = effectScope()
  const press = scope.run(() =>
    usePressDrag<string, Position>({
      getThreshold: () => threshold,
      getClock: () => clock,
      getLandingAt: (_item, at) => (at.x < 500 ? at : null),
      settle,
      begin,
    }),
  )!

  return {
    ...press,
    settle,
    begin,
    scope,
    frame: () => {
      const run = next
      next = null
      run?.(0)
    },
  }
}

describe('usePressDrag', () => {
  it('holds what was pressed without calling it a drag', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))

    expect(press.dragging.value).toEqual({ item: 'a row', moved: false })
    expect(press.position.value).toBeNull()
    expect(press.begin).not.toHaveBeenCalled()
    press.scope.stop()
  })

  it('stays a press until the pointer passes the threshold', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))

    window.dispatchEvent(pointer('pointermove', 13, 13))
    expect(press.dragging.value?.moved).toBe(false)
    expect(press.at.value).toBeNull()

    window.dispatchEvent(pointer('pointermove', 20, 10))
    expect(press.dragging.value?.moved).toBe(true)
    expect(press.position.value).toEqual({ x: 20, y: 10 })
    expect(press.at.value).toEqual({ x: 20, y: 10 })
    press.scope.stop()
  })

  it('says a drag began once, however far it travels', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))

    window.dispatchEvent(pointer('pointermove', 40, 10))
    window.dispatchEvent(pointer('pointermove', 80, 10))

    expect(press.begin).toHaveBeenCalledExactlyOnceWith('a row')
    press.scope.stop()
  })

  it('settles on the landing the pointer was over', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointerup', 40, 60))

    expect(press.settle).toHaveBeenCalledExactlyOnceWith('a row', { x: 40, y: 60 })
    expect(press.at.value).toBeNull()
    expect(press.position.value).toBeNull()
    press.scope.stop()
  })

  it('settles on nothing where the pointer is over no landing', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 600, 60))
    window.dispatchEvent(pointer('pointerup', 600, 60))

    expect(press.settle).toHaveBeenCalledExactlyOnceWith('a row', null)
    press.scope.stop()
  })

  it('settles nothing where the press never travelled', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointerup', 11, 11))

    expect(press.settle).not.toHaveBeenCalled()
    press.scope.stop()
  })

  it('holds what was carried one frame past the release', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointerup', 40, 60))

    expect(press.dragging.value?.moved).toBe(true)
    press.frame()
    expect(press.dragging.value).toBeNull()
    press.scope.stop()
  })

  it('lets go of the window when the release comes', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointerup', 40, 60))
    press.frame()

    window.dispatchEvent(pointer('pointermove', 90, 90))
    expect(press.dragging.value).toBeNull()
    expect(press.settle).toHaveBeenCalledTimes(1)
    press.scope.stop()
  })

  it('lets go of the window when the scope goes', () => {
    const press = createPress()
    press.lift('a row', pointer('pointerdown', 10, 10))
    press.scope.stop()

    window.dispatchEvent(pointer('pointermove', 40, 60))
    expect(press.dragging.value).toBeNull()
    expect(press.begin).not.toHaveBeenCalled()
  })
})
