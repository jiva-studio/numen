import { describe, expect, it } from 'vitest'

import { createRequestGuard } from './requestGuard'

describe('the request whose response is processed', () => {
  it('is the one requested, where nothing was requested after it', () => {
    const guard = createRequestGuard()
    expect(guard.acquireRequest().isCurrent).toBe(true)
  })

  it('is the last of several, and none of the ones before it', () => {
    const guard = createRequestGuard()
    const first = guard.acquireRequest()
    const second = guard.acquireRequest()
    const third = guard.acquireRequest()
    expect([first.isCurrent, second.isCurrent, third.isCurrent]).toStrictEqual([false, false, true])
  })

  it('is none, where what was on its way was cancelled', () => {
    const guard = createRequestGuard()
    const one = guard.acquireRequest()
    guard.cancel()
    expect(one.isCurrent).toBe(false)
  })

  it('is the one requested after cancel', () => {
    const guard = createRequestGuard()
    guard.acquireRequest()
    guard.cancel()
    expect(guard.acquireRequest().isCurrent).toBe(true)
  })

  it('is none once it has closed, whenever a response lands', () => {
    const guard = createRequestGuard()
    const one = guard.acquireRequest()
    guard.close()
    expect(one.isCurrent).toBe(false)
  })

  it('is none for a request initiated after it closed', () => {
    const guard = createRequestGuard()
    guard.close()
    expect(guard.acquireRequest().isCurrent).toBe(false)
  })
})

describe('the response that arrives', () => {
  it('is processed where nothing newer has been processed yet', () => {
    const guard = createRequestGuard()
    const first = guard.acquireRequest()
    guard.acquireRequest()
    expect(first.claim()).toBe(true)
  })

  it('is discarded where something newer has been processed already', () => {
    const guard = createRequestGuard()
    const first = guard.acquireRequest()
    const second = guard.acquireRequest()

    expect(second.claim()).toBe(true)
    expect(first.claim()).toBe(false)
  })

  it('is processed again by the one that claimed it, for multi-part responses', () => {
    const guard = createRequestGuard()
    const one = guard.acquireRequest()

    expect(one.claim()).toBe(true)
    expect(one.claim()).toBe(true)
  })

  it('is discarded where what was on its way was cancelled', () => {
    const guard = createRequestGuard()
    const one = guard.acquireRequest()
    guard.cancel()
    expect(one.claim()).toBe(false)
  })

  it('is discarded once it has closed', () => {
    const guard = createRequestGuard()
    const one = guard.acquireRequest()
    guard.close()
    expect(one.claim()).toBe(false)
  })
})

describe('whether requests are still being processed', () => {
  it('is true until it closes', () => {
    const guard = createRequestGuard()
    expect(guard.open()).toBe(true)
    guard.close()
    expect(guard.open()).toBe(false)
  })

  it('remains through cancel', () => {
    const guard = createRequestGuard()
    guard.cancel()
    expect(guard.open()).toBe(true)
  })
})
