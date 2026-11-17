import { describe, expect, it, vi } from 'vitest'
import { shallowRef, watch } from 'vue'

import { createRemoteValues } from './remoteValues'

interface Answer {
  readonly version: bigint
  readonly value: boolean
}

const createDeferred = <T>() => {
  let resolve: (value: T) => void = () => undefined
  let reject: (error: unknown) => void = () => undefined
  const promise = new Promise<T>((yes, no) => {
    resolve = yes
    reject = no
  })
  return { promise, resolve, reject }
}

const createSwitch = (confirmed: boolean) => {
  const remote = shallowRef(confirmed)
  const batches: {
    values: ReadonlyMap<string, boolean>
    answer: ReturnType<typeof createDeferred<Answer>>
  }[] = []
  const values = createRemoteValues<boolean, Answer>(
    (batch) => {
      const answer = createDeferred<Answer>()
      batches.push({ values: batch, answer })
      return answer.promise
    },
    (answer) => {
      remote.value = answer.value
    },
  )
  const history: boolean[] = []
  const getShownValue = () => values.getRequestedValue('on')?.value ?? remote.value
  watch(getShownValue, (value) => history.push(value), { flush: 'sync' })
  /** A read answered at `version` with `value`, taken or dropped as the answers taken before say. */
  const read = (version: bigint, value: boolean) =>
    values.takeAnswer(version, () => (remote.value = value))
  return { remote, values, batches, history, getShownValue, read }
}

describe('a switch bound to a remote value', () => {
  it('shows the requested value until its write answers, and then the value the write answered', async () => {
    const one = createSwitch(false)
    expect(one.values.isPending('on')).toBe(false)
    const write = one.values.requestWrite(new Map([['on', true]]))
    expect(one.values.isPending('on')).toBe(true)
    expect(one.getShownValue()).toBe(true)
    one.batches[0]?.answer.resolve({ version: 1n, value: true })
    await write
    expect(one.values.isPending('on')).toBe(false)
    expect(one.values.getRequestedValue('on')).toBeUndefined()
    expect(one.remote.value).toBe(true)
    expect(one.history).toEqual([true])
  })

  it('drops a read that began before a write and answers after it', async () => {
    const one = createSwitch(false)
    const write = one.values.requestWrite(new Map([['on', true]]))
    one.batches[0]?.answer.resolve({ version: 2n, value: true })
    await write
    expect(one.read(1n, false)).toBe(false)
    expect(one.remote.value).toBe(true)
    expect(one.getShownValue()).toBe(true)
    expect(one.history).toEqual([true])
  })

  it('takes a read of the version already taken or a newer one', async () => {
    const one = createSwitch(false)
    const write = one.values.requestWrite(new Map([['on', true]]))
    one.batches[0]?.answer.resolve({ version: 2n, value: true })
    await write
    expect(one.read(2n, true)).toBe(true)
    expect(one.read(3n, false)).toBe(true)
    expect(one.remote.value).toBe(false)
  })

  it('takes reads by their version, not by the order they answer', () => {
    const one = createSwitch(false)
    expect(one.read(2n, true)).toBe(true)
    expect(one.read(1n, false)).toBe(false)
    expect(one.remote.value).toBe(true)
  })

  it('shows the confirmed value again where the write failed', async () => {
    const one = createSwitch(false)
    const write = one.values.requestWrite(new Map([['on', true]]))
    one.batches[0]?.answer.reject(new Error('refused'))
    await expect(write).rejects.toThrow('refused')
    expect(one.history).toEqual([true, false])
    expect(one.values.isPending('on')).toBe(false)
  })

  it('sends the newest value requested during a write after it, once', async () => {
    const one = createSwitch(false)
    const first = one.values.requestWrite(new Map([['on', true]]))
    const second = one.values.requestWrite(new Map([['on', false]]))
    const third = one.values.requestWrite(new Map([['on', true]]))
    expect(one.batches).toHaveLength(1)
    one.batches[0]?.answer.resolve({ version: 1n, value: true })
    await first
    expect(one.values.isPending('on')).toBe(true)
    await vi.waitFor(() => expect(one.batches).toHaveLength(2))
    expect(one.batches.map((write) => write.values.get('on'))).toEqual([true, true])
    one.batches[1]?.answer.resolve({ version: 2n, value: true })
    await Promise.all([second, third])
    expect(one.history).toEqual([true, false, true])
    expect(one.values.isPending('on')).toBe(false)
  })

  it('keeps a later request where an earlier write fails', async () => {
    const one = createSwitch(true)
    const first = one.values.requestWrite(new Map([['on', true]]))
    const second = one.values.requestWrite(new Map([['on', false]]))
    one.batches[0]?.answer.reject(new Error('refused'))
    await expect(first).rejects.toThrow('refused')
    expect(one.getShownValue()).toBe(false)
    await vi.waitFor(() => expect(one.batches).toHaveLength(2))
    one.batches[1]?.answer.resolve({ version: 1n, value: false })
    await second
    expect(one.history).toEqual([false])
  })
})
