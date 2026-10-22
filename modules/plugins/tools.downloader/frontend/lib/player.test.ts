/**
 * What a frame is told to play from a moment.
 */
import { describe, expect, it } from 'vitest'
import { createSeekMessage, getFrameOrigin, seekFrame } from './player'

describe('the message that plays from a moment', () => {
  it('is a command to seek, in seconds', () => {
    expect(JSON.parse(createSeekMessage(83_000))).toStrictEqual({
      event: 'command',
      func: 'seekTo',
      args: [83, true],
    })
  })
})

describe('where a frame is told', () => {
  it('is the origin of its address', () => {
    expect(getFrameOrigin('http://127.0.0.1:9/token/embed/x')).toBe('http://127.0.0.1:9')
  })

  it('is nowhere for an address that is not absolute', () => {
    expect(getFrameOrigin('not-a-url')).toBeNull()
  })
})

describe('a frame told to play from a moment', () => {
  it('is sent the message at its origin', () => {
    const dispatched: [string, string][] = []
    const target = {
      postMessage: (message: string, origin: string) => dispatched.push([message, origin]),
    } as unknown as Window

    seekFrame(target, 'http://127.0.0.1:9/token/embed/x', 1_500)

    expect(dispatched).toStrictEqual([[createSeekMessage(1_500), 'http://127.0.0.1:9']])
  })

  it('is sent nothing where the address has no origin or there is no frame', () => {
    const dispatched: unknown[] = []
    const target = { postMessage: (...args: unknown[]) => dispatched.push(args) } as unknown as Window

    seekFrame(target, 'not-a-url', 1_000)
    seekFrame(null, 'http://127.0.0.1:9/x', 1_000)

    expect(dispatched).toStrictEqual([])
  })
})
