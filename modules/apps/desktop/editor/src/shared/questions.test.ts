import { describe, expect, it } from 'vitest'

import { answerGuard } from './questions'

describe('the question whose answer is drawn', () => {
  it('is the one asked, where nothing was asked after it', () => {
    const asks = answerGuard()
    expect(asks.ask().current).toBe(true)
  })

  it('is the last of several, and none of the ones before it', () => {
    const asks = answerGuard()
    const first = asks.ask()
    const second = asks.ask()
    const third = asks.ask()
    expect([first.current, second.current, third.current]).toStrictEqual([false, false, true])
  })

  it('is none, where what was on its way was let go of', () => {
    const asks = answerGuard()
    const one = asks.ask()
    asks.drop()
    expect(one.current).toBe(false)
  })

  it('is the one asked after the drop', () => {
    const asks = answerGuard()
    asks.ask()
    asks.drop()
    expect(asks.ask().current).toBe(true)
  })

  it('is none once it has closed, whenever an answer lands', () => {
    const asks = answerGuard()
    const one = asks.ask()
    asks.close()
    expect(one.current).toBe(false)
  })

  it('is none for a question asked after it closed', () => {
    const asks = answerGuard()
    asks.close()
    expect(asks.ask().current).toBe(false)
  })
})

describe('the answer that lands', () => {
  // Two answers to two questions can arrive in either order, and the older of
  // them carries less than the newer.
  it('is drawn where nothing newer has been drawn yet', () => {
    const asks = answerGuard()
    const first = asks.ask()
    asks.ask()
    expect(first.claim()).toBe(true)
  })

  it('is let go of where something newer has been drawn already', () => {
    const asks = answerGuard()
    const first = asks.ask()
    const second = asks.ask()

    expect(second.claim()).toBe(true)
    expect(first.claim()).toBe(false)
  })

  it('is drawn again by the one that drew it, which is one answer arriving in parts', () => {
    const asks = answerGuard()
    const one = asks.ask()

    expect(one.claim()).toBe(true)
    expect(one.claim()).toBe(true)
  })

  it('is let go of where what was on its way was let go of', () => {
    const asks = answerGuard()
    const one = asks.ask()
    asks.drop()
    expect(one.claim()).toBe(false)
  })

  it('is let go of once it has closed', () => {
    const asks = answerGuard()
    const one = asks.ask()
    asks.close()
    expect(one.claim()).toBe(false)
  })
})

describe('whether answers are still being drawn', () => {
  it('is so until it closes', () => {
    const asks = answerGuard()
    expect(asks.open()).toBe(true)
    asks.close()
    expect(asks.open()).toBe(false)
  })

  it('stands through a drop', () => {
    const asks = answerGuard()
    asks.drop()
    expect(asks.open()).toBe(true)
  })
})
