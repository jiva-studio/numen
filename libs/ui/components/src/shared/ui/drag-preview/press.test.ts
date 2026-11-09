import { effectScope } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { usePressDrag } from './press'
import type { Position } from '@/shared/lib/math/geometry'
import type { Clock } from '@/shared/lib/time/clock'

const pointer = (type: string, x: number, y: number, pointerId = 1) =>
  new PointerEvent(type, { clientX: x, clientY: y, pointerId, bubbles: true })

function createCapturingElement() {
  const element = document.createElement('div')
  const captured = new Set<number>()
  element.setPointerCapture = (id) => {
    captured.add(id)
  }
  element.releasePointerCapture = (id) => {
    captured.delete(id)
  }
  element.hasPointerCapture = (id) => captured.has(id)
  document.body.append(element)
  return { element, captured }
}

function createPress(threshold = 4, withBegin = true) {
  const commit = vi.fn<(item: string, target: Position | null, point: Position) => void>()
  const begin = vi.fn<(item: string) => void>()
  const cancel = vi.fn<(item: string) => void>()
  let next: ((now: number) => void) | null = null
  let scheduled = 0

  const clock: Clock = {
    now: () => 0,
    schedule: (run) => {
      scheduled++
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
      getDropTargetAt: (_item, index) => (index.x < 500 ? index : null),
      commit,
      ...(withBegin ? { begin, cancel } : {}),
    }),
  )!

  return {
    ...press,
    commit,
    begin,
    cancel,
    scope,
    getScheduled: () => scheduled,
    frame: () => {
      const run = next
      next = null
      run?.(0)
    },
  }
}

describe('usePressDrag', () => {
  it('records the pressed item before the drag starts', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))

    expect(press.dragging.value).toEqual({ item: 'a row', hasMoved: false })
    expect(press.position.value).toBeNull()
    expect(press.begin).not.toHaveBeenCalled()
    press.scope.stop()
  })

  it('starts a drag after the pointer moves past the threshold', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))

    window.dispatchEvent(pointer('pointermove', 13, 13))
    expect(press.dragging.value?.hasMoved).toBe(false)
    expect(press.at.value).toBeNull()

    window.dispatchEvent(pointer('pointermove', 20, 10))
    expect(press.dragging.value?.hasMoved).toBe(true)
    expect(press.position.value).toEqual({ x: 20, y: 10 })
    expect(press.at.value).toEqual({ x: 20, y: 10 })
    press.scope.stop()
  })

  it('calls begin once when the drag starts', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))

    window.dispatchEvent(pointer('pointermove', 40, 10))
    window.dispatchEvent(pointer('pointermove', 80, 10))

    expect(press.begin).toHaveBeenCalledExactlyOnceWith('a row')
    press.scope.stop()
  })

  it('commits the drop target on pointerup', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointerup', 40, 60))

    expect(press.commit).toHaveBeenCalledExactlyOnceWith(
      'a row',
      { x: 40, y: 60 },
      { x: 40, y: 60 },
    )
    expect(press.at.value).toBeNull()
    expect(press.position.value).toBeNull()
    press.scope.stop()
  })

  it('commits a null target when pointerup is outside every target', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 600, 60))
    window.dispatchEvent(pointer('pointerup', 600, 60))

    expect(press.commit).toHaveBeenCalledExactlyOnceWith('a row', null, { x: 600, y: 60 })
    press.scope.stop()
  })

  it('does not commit when the pointer moves below the threshold', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointerup', 11, 11))

    expect(press.commit).not.toHaveBeenCalled()
    press.scope.stop()
  })

  it('clears dragging on the frame after pointerup', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointerup', 40, 60))

    expect(press.dragging.value?.hasMoved).toBe(true)
    press.frame()
    expect(press.dragging.value).toBeNull()
    press.scope.stop()
  })

  it('ignores pointermove after pointerup', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointerup', 40, 60))
    press.frame()

    window.dispatchEvent(pointer('pointermove', 90, 90))
    expect(press.dragging.value).toBeNull()
    expect(press.commit).toHaveBeenCalledTimes(1)
    press.scope.stop()
  })

  it('removes listeners when the scope stops', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    press.scope.stop()

    window.dispatchEvent(pointer('pointermove', 40, 60))
    expect(press.dragging.value).toBeNull()
    expect(press.begin).not.toHaveBeenCalled()
  })

  it('does not start a drag when the pointer moves to the threshold', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))

    window.dispatchEvent(pointer('pointermove', 14, 10))
    expect(press.dragging.value?.hasMoved).toBe(false)

    window.dispatchEvent(pointer('pointermove', 10, 14))
    expect(press.dragging.value?.hasMoved).toBe(false)

    window.dispatchEvent(pointer('pointermove', 10, 15))
    expect(press.dragging.value?.hasMoved).toBe(true)
    press.scope.stop()
  })

  it('starts a drag when the pointer moves horizontally past the threshold', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))

    window.dispatchEvent(pointer('pointermove', 15, 10))

    expect(press.dragging.value?.hasMoved).toBe(true)
    press.scope.stop()
  })

  it('starts a drag when begin is undefined', () => {
    const press = createPress(4, false)
    const faults: unknown[] = []
    const onError = (event: ErrorEvent) => {
      faults.push(event.error)
    }
    window.addEventListener('error', onError)
    press.startPress('a row', pointer('pointerdown', 10, 10))

    window.dispatchEvent(pointer('pointermove', 40, 10))
    window.removeEventListener('error', onError)

    expect(faults).toStrictEqual([])
    expect(press.dragging.value?.hasMoved).toBe(true)
    press.scope.stop()
  })

  it('commits the drop target at the pointerup position', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointerup', 700, 60))

    expect(press.commit).toHaveBeenCalledExactlyOnceWith('a row', null, { x: 700, y: 60 })
    press.scope.stop()
  })

  it('calls cancel and does not commit on pointercancel', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointercancel', 40, 60))

    expect(press.commit).not.toHaveBeenCalled()
    expect(press.cancel).toHaveBeenCalledExactlyOnceWith('a row')
    expect(press.at.value).toBeNull()
    expect(press.position.value).toBeNull()
    press.frame()
    expect(press.dragging.value).toBeNull()
    press.scope.stop()
  })

  it('does not call cancel for a press below the threshold', () => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointercancel', 11, 11))

    expect(press.cancel).not.toHaveBeenCalled()
    expect(press.commit).not.toHaveBeenCalled()
    press.scope.stop()
  })

  it('cancels a drag when cancel is undefined', () => {
    const press = createPress(4, false)
    press.startPress('a row', pointer('pointerdown', 10, 10))
    window.dispatchEvent(pointer('pointermove', 40, 60))
    window.dispatchEvent(pointer('pointercancel', 40, 60))

    expect(press.commit).not.toHaveBeenCalled()
    expect(press.position.value).toBeNull()
    press.scope.stop()
  })

  it.each(['pointerup', 'pointercancel'])(
    'captures the pointer on the pressed element until %s',
    (kind) => {
      const press = createPress()
      const { element, captured } = createCapturingElement()
      element.addEventListener('pointerdown', (event) => press.startPress('a row', event))
      element.dispatchEvent(pointer('pointerdown', 10, 10, 7))

      expect(captured).toStrictEqual(new Set([7]))
      window.dispatchEvent(pointer('pointermove', 40, 60, 7))
      expect(captured).toStrictEqual(new Set([7]))

      window.dispatchEvent(pointer(kind, 40, 60, 7))
      expect(captured).toStrictEqual(new Set())
      element.remove()
      press.scope.stop()
    },
  )

  it('releases pointer capture when the scope stops', () => {
    const press = createPress()
    const { element, captured } = createCapturingElement()
    element.addEventListener('pointerdown', (event) => press.startPress('a row', event))
    element.dispatchEvent(pointer('pointerdown', 10, 10, 7))
    press.scope.stop()

    expect(captured).toStrictEqual(new Set())
    element.remove()
  })

  it('commits a drag when setPointerCapture throws', () => {
    const press = createPress()
    const element = document.createElement('div')
    element.setPointerCapture = () => {
      throw new DOMException('No active pointer', 'NotFoundError')
    }
    element.addEventListener('pointerdown', (event) => press.startPress('a row', event))
    element.dispatchEvent(pointer('pointerdown', 10, 10, 7))
    window.dispatchEvent(pointer('pointermove', 40, 60, 7))
    window.dispatchEvent(pointer('pointerup', 40, 60, 7))

    expect(press.commit).toHaveBeenCalledOnce()
    press.scope.stop()
  })

  it.each(['pointermove', 'pointerup', 'pointercancel'])(
    'ignores %s after pointerup before the next frame',
    (kind) => {
      const press = createPress()
      press.startPress('a row', pointer('pointerdown', 10, 10))
      window.dispatchEvent(pointer('pointermove', 40, 60))
      window.dispatchEvent(pointer('pointerup', 40, 60))

      window.dispatchEvent(pointer(kind, 90, 90))

      expect(press.commit).toHaveBeenCalledTimes(1)
      expect(press.position.value).toBeNull()
      expect(press.at.value).toBeNull()
      expect(press.getScheduled()).toBe(1)
      press.scope.stop()
    },
  )

  it.each(['pointerup', 'pointercancel'])('ignores %s after the scope stops', (kind) => {
    const press = createPress()
    press.startPress('a row', pointer('pointerdown', 10, 10))
    press.scope.stop()

    window.dispatchEvent(pointer(kind, 40, 60))

    expect(press.getScheduled()).toBe(0)
    expect(press.commit).not.toHaveBeenCalled()
  })
})
