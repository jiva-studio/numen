import { describe, expect, it } from 'vitest'

import { createWriteOrder } from './remoteValues'

describe('the order of reads and writes', () => {
  it('says a read overlapped a write where one was in flight when it began or finished since', () => {
    const order = createWriteOrder()
    expect(order.isWriteInFlight()).toBe(false)
    const before = order.beginRead()
    expect(before.isWriting).toBe(false)
    const settleWrite = order.beginWrite()
    expect(order.isWriteInFlight()).toBe(true)
    expect(order.isReadAfterWrites(before)).toBe(false)
    const during = order.beginRead()
    expect(during.isWriting).toBe(true)
    expect(order.isReadAfterWrites(during)).toBe(false)
    settleWrite()
    const after = order.beginRead()
    expect(after.isWriting).toBe(false)
    expect([before, during, after].map(order.isReadAfterWrites)).toEqual([false, false, true])
    expect(order.isReadAfterWrites({ place: 99, settledWrites: 1, isWriting: true })).toBe(false)
  })

  it('settles a write once and accepts a read at the same place', () => {
    const order = createWriteOrder()
    const settleWrite = order.beginWrite()
    settleWrite()
    settleWrite()
    expect(order.getSettledWrites()).toBe(1)
    expect(order.isWriteInFlight()).toBe(false)
    const read = order.beginRead()
    expect(order.takeRead(read)).toBe(true)
    expect(order.takeRead(read)).toBe(true)
  })
})
