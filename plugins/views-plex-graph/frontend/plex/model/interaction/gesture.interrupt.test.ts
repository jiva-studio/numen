/** What interrupts a gesture: a context menu, and a second press while one is under way. */
import { effectScope, type EffectScope } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { usePlexGesture } from './gesture'
import { arrangePlex, DEFAULT_OPTIONS } from '../../lib/arrange'
import { build } from '../../fixtures/build'

const frame = arrangePlex(build('A node', { parent: 1, child: 2 }))

const scopes: EffectScope[] = []

const setUp = () => {
  const scope = effectScope()
  scopes.push(scope)
  const gesture = scope.run(() =>
    usePlexGesture({
      frame: () => frame,
      options: () => DEFAULT_OPTIONS,
      relations: () => ['child'],
      threshold: () => 10,
      commit: vi.fn(),
    }),
  )!
  return gesture
}

const drawing = () => ({
  setPointerCapture: vi.fn(),
  releasePointerCapture: vi.fn(),
  getScreenCTM: () => ({ inverse: () => ({ a: 1, b: 0, c: 0, d: 1, e: 0, f: 0 }) }),
})

const pressOn = (surface: ReturnType<typeof drawing>, pointerId: number) =>
  ({
    target: { ownerSVGElement: surface },
    clientX: 1,
    clientY: 2,
    pointerId,
  }) as unknown as PointerEvent

afterEach(() => {
  for (const scope of scopes.splice(0)) scope.stop()
  vi.restoreAllMocks()
})

describe('a context menu while a gesture is under way', () => {
  it('cancels the gesture and keeps the menu from opening', () => {
    const gesture = setUp()
    gesture.begin('focus', pressOn(drawing(), 1))
    const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true })

    window.dispatchEvent(event)

    expect(gesture.from.value).toBeNull()
    expect(event.defaultPrevented).toBe(true)
  })

  it('is left alone while no gesture is under way', () => {
    const gesture = setUp()
    const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true })

    window.dispatchEvent(event)

    expect(gesture.from.value).toBeNull()
    expect(event.defaultPrevented).toBe(false)
  })
})

describe('a second press while a gesture is under way', () => {
  it('lets go of the listeners of the first pointer', () => {
    const add = vi.spyOn(window, 'addEventListener')
    const remove = vi.spyOn(window, 'removeEventListener')
    const gesture = setUp()

    gesture.begin('focus', pressOn(drawing(), 1))
    gesture.begin('focus', pressOn(drawing(), 2))
    gesture.cancel()

    expect(add.mock.calls.filter(([type]) => type === 'pointermove')).toHaveLength(2)
    expect(remove.mock.calls.filter(([type]) => type === 'pointermove')).toHaveLength(2)
  })

  it('releases the first pointer', () => {
    const gesture = setUp()
    const first = drawing()

    gesture.begin('focus', pressOn(first, 1))
    gesture.begin('focus', pressOn(drawing(), 2))

    expect(first.releasePointerCapture).toHaveBeenCalledWith(1)
  })
})
