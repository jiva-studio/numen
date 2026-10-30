/**
 * A stream taken up again, asked without a core to talk to.
 *
 * The failure here looks like nothing at all: a window that stopped following
 * shows what the vault held a moment ago, with no error and no way back.
 */
import { describe, expect, it, vi } from 'vitest'
import { createFollower, AGAIN, LOST } from './stream'

/** A window that is open for as many streams as the test allows it. */
const window = (streams = 3) => {
  const waits: number[] = []
  const missing: string[] = []
  let acquiredCount = 0
  const follows = createFollower({
    isOpen: () => acquiredCount <= streams,
    setLost: (response) => missing.push(response),
    wait: async (ms) => {
      acquiredCount++
      waits.push(ms)
    },
  })
  return { follows, waits, lost: missing, took: () => acquiredCount }
}

/** A stream that says what it was given and ends. */
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
    expect(one.waits).toStrictEqual([AGAIN, AGAIN, AGAIN])
  })

  it('waits as long as it was told to', async () => {
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

    expect(one.lost[0]).toBe(LOST)
  })

  it('says nothing about it once it has it back', async () => {
    const one = window(1)

    await one.follows(() => {
      throw new Error('connection lost')
    }, vi.fn())

    expect(one.lost[1]).toBe('')
  })

  it('is taken up again when what answers it throws', async () => {
    const one = window(1)
    let reply = 0

    await one.follows(createStream('a change'), () => {
      reply++
      throw new Error('the answer went wrong')
    })

    expect(reply).toBeGreaterThanOrEqual(2)
    expect(one.lost[0]).toBe(LOST)
  })
})

describe('a window that has closed', () => {
  it('follows nothing, and waits for nothing', async () => {
    const closed = createFollower({
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
    const follows = createFollower({
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
    const follows = createFollower({
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
    const follows = createFollower({
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
      throw new Error('the connection went')
    }, vi.fn())

    expect(order).toStrictEqual(['lost', 'resets'])
  })
})

describe('a window that closes while a stream is read', () => {
  it('waits for nothing and lets go of nothing once it closed on an item', async () => {
    let open = true
    const waits: number[] = []
    const resets: number[] = []
    const follows = createFollower({
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

  it('says nothing, waits for nothing and lets go of nothing when the stream fails after it closed', async () => {
    let open = true
    const missing: string[] = []
    const waits: number[] = []
    const resets: number[] = []
    const follows = createFollower({
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
      open = false
      throw new Error('the connection went')
    }, vi.fn())

    expect(missing).toStrictEqual([])
    expect(waits).toStrictEqual([])
    expect(resets).toStrictEqual([])
  })
})
