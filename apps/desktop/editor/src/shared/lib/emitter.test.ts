import { describe, expect, it, vi } from 'vitest'
import { createEmitter } from './emitter'

describe('an emitter', () => {
  it('calls every listener with the emitted value', () => {
    const emitter = createEmitter<string>()
    const first = vi.fn()
    const second = vi.fn()
    emitter.subscribe(first)
    emitter.subscribe(second)

    emitter.emit('Coral.md')

    expect(first).toHaveBeenCalledExactlyOnceWith('Coral.md')
    expect(second).toHaveBeenCalledExactlyOnceWith('Coral.md')
  })

  it('calls a listener only between its subscription and its stop', () => {
    const emitter = createEmitter<number>()
    const listener = vi.fn()
    emitter.emit(1)

    const stop = emitter.subscribe(listener)
    emitter.emit(2)
    stop()
    emitter.emit(3)

    expect(listener.mock.calls).toStrictEqual([[2]])
  })
})
