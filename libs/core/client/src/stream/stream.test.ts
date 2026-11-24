/**
 * A resumed stream, tested without a core to talk to.
 *
 * The failure here looks like nothing at all: a window that stopped following
 * shows what the source held a moment ago, with no error and no way back.
 */
import { describe, expect, it, vi } from 'vitest'
import { createStreamReader } from './stream'
import { RETRY_DELAY_MS, CONNECTION_LOST_MESSAGE } from './streamRetry'

const window = (streams = 3) => {
  const waits: number[] = []
  const missing: string[] = []
  let acquiredCount = 0
  const follows = createStreamReader({
    isOpen: () => acquiredCount <= streams,
    setLost: (response) => missing.push(response),
    wait: async (ms) => {
      acquiredCount++
      waits.push(ms)
    },
  })
  return { follows, waits, lost: missing, took: () => acquiredCount }
}

const createStream = <T>(...response: readonly T[]) =>
  async function* () {
    for (const one of response) yield one
  }

describe('a stream that ends', () => {
  it('is taken up again, after a wait each time', async () => {
    const one = window(2)
    const received: string[] = []

    await one.follows(createStream('a change'), (response) => {
      received.push(response)
    })

    expect(received.length).toBeGreaterThanOrEqual(2)
    expect(one.waits).toStrictEqual([RETRY_DELAY_MS, RETRY_DELAY_MS, RETRY_DELAY_MS])
  })

  it('waits as long as it was configured to', async () => {
    const one = window(1)

    await one.follows(createStream('a change'), vi.fn(), 50)

    expect(one.waits).toStrictEqual([50, 50])
  })
})

describe('a stream that fails', () => {
  it('says what it lost touch with, and takes it up again', async () => {
    const one = window(1)

    await one.follows(() => {
      throw new Error('connection lost')
    }, vi.fn())

    expect(one.lost[0]).toBe(CONNECTION_LOST_MESSAGE)
  })

  it('keeps the loss through failed retries and clears it after handling the first recovered item', async () => {
    const one = window(3)
    let attempts = 0
    const observed: string[][] = []
    await one.follows(
      async function* () {
        if (++attempts < 3) throw new Error('offline')
        yield 'connected'
      },
      () => {
        observed.push([...one.lost])
      },
    )

    expect(observed).toEqual([[CONNECTION_LOST_MESSAGE], [CONNECTION_LOST_MESSAGE, '']])
    expect(one.lost).toEqual([CONNECTION_LOST_MESSAGE, ''])
  })

  it('keeps the loss while another subscription is disconnected', async () => {
    let open = true
    const reports: string[] = []
    const retries: (() => void)[] = []
    const follow = createStreamReader({
      isOpen: () => open,
      setLost: (text) => reports.push(text),
      wait: () => new Promise((resolve) => retries.push(resolve)),
    })
    const reconnect = () => {
      let attempt = 0
      return follow(async function* () {
        if (++attempt === 1) throw new Error('offline')
        yield 'connected'
      }, vi.fn())
    }
    const subscriptions = [reconnect(), reconnect()]
    await vi.waitFor(() => expect(retries).toHaveLength(2))
    retries[0]!()
    await vi.waitFor(() => expect(retries).toHaveLength(3))
    expect(reports).toEqual([CONNECTION_LOST_MESSAGE])
    retries[1]!()
    await vi.waitFor(() => expect(retries).toHaveLength(4))
    expect(reports).toEqual([CONNECTION_LOST_MESSAGE, ''])
    open = false
    retries.slice(2).forEach((resume) => resume())
    await Promise.all(subscriptions)
  })

  it('is taken up again when what answers it throws', async () => {
    const one = window(1)
    let reply = 0

    await one.follows(createStream('a change'), () => {
      reply++
      throw new Error('the answer went wrong')
    })

    expect(reply).toBeGreaterThanOrEqual(2)
    expect(one.lost[0]).toBe(CONNECTION_LOST_MESSAGE)
  })
})

describe('a window that has closed', () => {
  it('follows nothing, and waits for nothing', async () => {
    const closed = createStreamReader({
      isOpen: () => false,
      setLost: vi.fn(),
      wait: () => Promise.resolve(),
    })
    const received: string[] = []

    await closed(createStream('a change'), (response) => {
      received.push(response)
    })

    expect(received).toStrictEqual([])
  })

  it('drops what arrives after it closed', async () => {
    let open = true
    const received: string[] = []
    const follows = createStreamReader({
      isOpen: () => open,
      setLost: vi.fn(),
      wait: () => Promise.resolve(),
    })

    await follows(createStream('one', 'two'), (response) => {
      received.push(response)
      open = false
    })

    expect(received).toStrictEqual(['one'])
  })
})

describe('what a follower holds for one reading of a stream', () => {
  it('is release when the stream ends, before it is taken up again', async () => {
    const order: string[] = []
    let acquiredCount = 0
    const follows = createStreamReader({
      isOpen: () => acquiredCount <= 2,
      setLost: vi.fn(),
      wait: async () => {
        acquiredCount++
        order.push('waits')
      },
      reset: () => {
        order.push('resets')
      },
    })

    await follows(createStream('a change'), vi.fn())

    expect(order).toStrictEqual(['resets', 'waits', 'resets', 'waits', 'resets', 'waits'])
  })

  it('is release when the stream fails, the same way', async () => {
    const order: string[] = []
    let acquiredCount = 0
    const follows = createStreamReader({
      isOpen: () => acquiredCount < 1,
      setLost: () => {
        order.push('lost')
      },
      wait: async () => {
        acquiredCount++
      },
      reset: () => {
        order.push('resets')
      },
    })

    await follows(async function* (): AsyncIterable<string> {
      yield* []
      throw new Error('the connection went')
    }, vi.fn())

    expect(order).toStrictEqual(['lost', 'resets'])
  })
})

describe('a window that closes while a stream is read', () => {
  it('waits for nothing and releases nothing once it closed on an item', async () => {
    let open = true
    const waits: number[] = []
    const resets: number[] = []
    const follows = createStreamReader({
      isOpen: () => open,
      setLost: vi.fn(),
      wait: (ms) => {
        waits.push(ms)
        return Promise.resolve()
      },
      reset: () => {
        resets.push(1)
      },
    })

    await follows(createStream('one', 'two'), () => {
      open = false
    })

    expect(waits).toStrictEqual([])
    expect(resets).toStrictEqual([])
  })

  it('reports no loss, waits and resets nothing when the stream fails after closing', async () => {
    let open = true
    const missing: string[] = []
    const waits: number[] = []
    const resets: number[] = []
    const follows = createStreamReader({
      isOpen: () => open,
      setLost: (response) => missing.push(response),
      wait: (ms) => {
        waits.push(ms)
        return Promise.resolve()
      },
      reset: () => {
        resets.push(1)
      },
    })

    await follows(async function* (): AsyncIterable<string> {
      yield* []
      open = false
      throw new Error('the connection went')
    }, vi.fn())

    expect(missing).toStrictEqual([])
    expect(waits).toStrictEqual([])
    expect(resets).toStrictEqual([])
  })
})
