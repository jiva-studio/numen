/**
 * A stream taken up again, asked without a core to talk to.
 *
 * The failure here looks like nothing at all: a window that stopped following
 * shows what the vault held a moment ago, with no error and no way back.
 */
import { describe, expect, it } from 'vitest'
import { createFollower, AGAIN, LOST } from './stream'

/** A window that is open for as many streams as the test allows it. */
const window = (streams = 3) => {
  const waits: number[] = []
  const lost: string[] = []
  let taken = 0
  const follows = createFollower({
    isOpen: () => taken <= streams,
    setLost: (said) => lost.push(said),
    wait: async (ms) => {
      taken++
      waits.push(ms)
    },
  })
  return { follows, waits, lost, took: () => taken }
}

/** A stream that says what it was given and ends. */
const createStream = <T,>(...said: readonly T[]) =>
  async function* () {
    for (const one of said) yield one
  }

describe('a stream that ends', () => {
  it('is taken up again, after a wait each time', async () => {
    const one = window(2)
    const heard: string[] = []

    await one.follows(createStream('a change'), (said) => void heard.push(said))

    expect(heard.length).toBeGreaterThanOrEqual(2)
    expect(one.waits).toStrictEqual([AGAIN, AGAIN, AGAIN])
  })

  it('waits as long as it was told to', async () => {
    const one = window(1)

    await one.follows(createStream('a change'), () => {}, 50)

    expect(one.waits).toStrictEqual([50, 50])
  })
})

describe('a stream that fails', () => {
  it('says what it lost touch with, and takes it up again', async () => {
    const one = window(1)

    await one.follows(
      () => {
        throw new Error('connection lost')
      },
      () => {},
    )

    expect(one.lost[0]).toBe(LOST)
  })

  it('says nothing about it once it has it back', async () => {
    const one = window(1)

    await one.follows(
      () => {
        throw new Error('connection lost')
      },
      () => {},
    )

    expect(one.lost[1]).toBe('')
  })

  it('is taken up again when what answers it throws', async () => {
    const one = window(1)
    let answered = 0

    await one.follows(createStream('a change'), () => {
      answered++
      throw new Error('the answer went wrong')
    })

    expect(answered).toBeGreaterThanOrEqual(2)
    expect(one.lost[0]).toBe(LOST)
  })
})

describe('a window that has closed', () => {
  it('follows nothing, and waits for nothing', async () => {
    const closed = createFollower({ isOpen: () => false, setLost: () => {}, wait: async () => {} })
    const heard: string[] = []

    await closed(createStream('a change'), (said) => void heard.push(said))

    expect(heard).toStrictEqual([])
  })

  it('drops what arrives after it closed', async () => {
    let open = true
    const heard: string[] = []
    const follows = createFollower({ isOpen: () => open, setLost: () => {}, wait: async () => {} })

    await follows(createStream('one', 'two'), (said) => {
      heard.push(said)
      open = false
    })

    expect(heard).toStrictEqual(['one'])
  })
})

describe('what a follower holds for one reading of a stream', () => {
  it('is let go of when the stream ends, before it is taken up again', async () => {
    const order: string[] = []
    let taken = 0
    const follows = createFollower({
      isOpen: () => taken <= 2,
      setLost: () => {},
      wait: async () => {
        taken++
        order.push('waits')
      },
      reset: () => void order.push('resets'),
    })

    await follows(createStream('a change'), () => {})

    expect(order).toStrictEqual(['resets', 'waits', 'resets', 'waits', 'resets', 'waits'])
  })

  it('is let go of when the stream fails, the same way', async () => {
    const order: string[] = []
    let taken = 0
    const follows = createFollower({
      isOpen: () => taken < 1,
      setLost: () => void order.push('lost'),
      wait: async () => {
        taken++
      },
      reset: () => void order.push('resets'),
    })

    await follows(
      async function* (): AsyncIterable<string> {
        throw new Error('the connection went')
      },
      () => {},
    )

    expect(order).toStrictEqual(['lost', 'resets'])
  })
})
