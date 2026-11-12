import { describe, expect, it } from 'vitest'
import { shallowRef, watch } from 'vue'

import { createRemoteValues } from './remoteValues'

const createDeferred = () => {
  let resolve: () => void = () => undefined
  let reject: (error: unknown) => void = () => undefined
  const promise = new Promise<void>((yes, no) => {
    resolve = yes
    reject = no
  })
  return { promise, resolve, reject }
}

const createSwitch = (confirmed: boolean) => {
  const remote = shallowRef(confirmed)
  const batches: {
    values: ReadonlyMap<string, boolean>
    answer: ReturnType<typeof createDeferred>
  }[] = []
  const values = createRemoteValues<boolean>((batch) => {
    const answer = createDeferred()
    batches.push({ values: batch, answer })
    return answer.promise
  })
  const history: boolean[] = []
  const getShownValue = () => values.getRequestedValue('on')?.value ?? remote.value
  watch(getShownValue, (value) => history.push(value), { flush: 'sync' })
  /** Answers a read begun now with the value the remote holds now, once `answer` is called. */
  const beginRead = () => {
    const pendingRead = values.beginRead()
    const current = remote.value
    return (value = current) => values.takeRead(pendingRead, () => (remote.value = value))
  }
  return { remote, values, batches, history, getShownValue, beginRead }
}

const settle = () => new Promise((resolve) => setTimeout(resolve, 0))

describe('a switch bound to a remote value', () => {
  it('shows the requested value until a read begun after the write is taken', async () => {
    const one = createSwitch(false)
    expect(one.values.isPending('on')).toBe(false)
    expect(one.values.isWriteInFlight()).toBe(false)
    const before = one.beginRead()
    const write = one.values.requestWrite(new Map([['on', true]]))
    expect(one.values.isPending('on')).toBe(true)
    expect(one.values.isWriteInFlight()).toBe(true)
    const during = one.beginRead()
    expect(before()).toBe(false)
    expect(during()).toBe(false)
    one.batches[0]?.answer.resolve()
    await write
    expect(one.values.isPending('on')).toBe(false)
    expect(one.getShownValue()).toBe(true)
    expect(one.beginRead()(true)).toBe(true)
    expect(one.history).toEqual([true])
    expect(one.values.getRequestedValue('on')).toBeUndefined()
  })

  it('shows the confirmed value again where the write failed, and takes a read begun after', async () => {
    const one = createSwitch(false)
    const write = one.values.requestWrite(new Map([['on', true]]))
    one.batches[0]?.answer.reject(new Error('refused'))
    await expect(write).rejects.toThrow('refused')
    expect(one.history).toEqual([true, false])
    expect(one.beginRead()(true)).toBe(true)
    expect(one.remote.value).toBe(true)
  })

  it('sends the newest value requested during a write after it, once', async () => {
    const one = createSwitch(false)
    const first = one.values.requestWrite(new Map([['on', true]]))
    const second = one.values.requestWrite(new Map([['on', false]]))
    const third = one.values.requestWrite(new Map([['on', true]]))
    expect(one.batches).toHaveLength(1)
    one.batches[0]?.answer.resolve()
    await first
    expect(one.values.isPending('on')).toBe(true)
    expect(one.beginRead()(false)).toBe(true)
    expect(one.values.isPending('on')).toBe(true)
    await settle()
    expect(one.batches.map((write) => write.values.get('on'))).toEqual([true, true])
    one.batches[1]?.answer.resolve()
    await Promise.all([second, third])
    expect(one.history).toEqual([true, false, true])
    expect(one.values.isWriteInFlight()).toBe(false)
  })

  it('keeps a later request where an earlier write fails', async () => {
    const one = createSwitch(true)
    const first = one.values.requestWrite(new Map([['on', true]]))
    const second = one.values.requestWrite(new Map([['on', false]]))
    one.batches[0]?.answer.reject(new Error('refused'))
    await expect(first).rejects.toThrow('refused')
    expect(one.getShownValue()).toBe(false)
    expect(one.values.getRequestedValue('on')?.value).toBe(false)
    await settle()
    one.batches[1]?.answer.resolve()
    await second
    expect(one.history).toEqual([false])
  })

  it('takes reads in the order they began', () => {
    const one = createSwitch(false)
    const older = one.beginRead()
    const newer = one.beginRead()
    expect(newer(true)).toBe(true)
    expect(older(false)).toBe(false)
    expect(one.remote.value).toBe(true)
  })
})
