import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from './results'
import type { Event } from './tab'
import type { Notes } from './types'
import { createNoteQueue, convertErrorCode } from './queue'

/** Throws the rejection reason of a detached promise, so the test run reports it as an unhandled rejection. */
function rethrow(error: unknown): never {
  throw error
}

type ErrorCode = string

describe('what a failed call is reported as', () => {
  it.each<[ErrorCode, string]>([
    ['missing', 'unreadable'],
    ['alreadyExists', 'unreadable'],
    ['invalidName', 'unreadable'],
    ['tooLarge', 'tooLarge'],
    ['notANote', 'notANote'],
    ['notText', 'notText'],
    ['bodyUnwritable', 'bodyUnwritable'],
    ['unreadable', 'unreadable'],
    ['unreachable', 'unreachable'],
  ])('turns %s into %s', (from, expected) => {
    expect(convertErrorCode(from)).toBe(expected)
  })
})

describe('the queue of an open note', () => {
  const setup = (notes: Partial<Notes> = {}, hasTab = true) => {
    const turned: [string, Event][] = []
    const resumed: string[] = []
    const queue = createNoteQueue(
      {
        read: async () => asValue({ body: '' }),
        write: async () => asValue({ body: '' }),
        ...notes,
      },
      {
        dispatch: (id, event) => turned.push([id, event]),
        hasTab: () => hasTab,
        resumeClosing: (id) => resumed.push(id),
      },
    )
    return { queue, turned, resumed }
  }

  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('its timer', () => {
    it('fires once, after the time it was armed for', () => {
      const { queue, turned } = setup()

      queue.startTimer('a', 100)
      vi.advanceTimersByTime(99)
      expect(turned).toEqual([])
      vi.advanceTimersByTime(1)

      expect(turned).toEqual([['a', { kind: 'timerFired' }]])
    })

    it('is replaced where the note is armed again', () => {
      const { queue, turned } = setup()

      queue.startTimer('a', 100)
      queue.startTimer('a', 300)
      vi.advanceTimersByTime(150)
      expect(turned).toEqual([])
      vi.advanceTimersByTime(150)

      expect(turned).toEqual([['a', { kind: 'timerFired' }]])
    })

    it('is kept apart from the timer of another note', () => {
      const { queue, turned } = setup()

      queue.startTimer('a', 100)
      queue.startTimer('b', 100)
      queue.cancelTimer('a')
      vi.advanceTimersByTime(100)

      expect(turned).toEqual([['b', { kind: 'timerFired' }]])
    })

    it('does not fire once disarmed', () => {
      const { queue, turned } = setup()

      queue.startTimer('a', 100)
      queue.cancelTimer('a')
      vi.advanceTimersByTime(500)

      expect(turned).toEqual([])
    })
  })

  describe('flushing', () => {
    it('is done at once for a note with no tab, and asks nothing of it', async () => {
      const { queue, turned } = setup({}, false)

      await queue.flushPending('a')

      expect(turned).toEqual([])
    })

    it('asks the tab to flush and waits until it is resolved', async () => {
      const { queue, turned } = setup()
      let isFinished = false

      const flushed = queue.flushPending('a').then(() => (isFinished = true))
      await Promise.resolve()
      expect(turned).toEqual([['a', { kind: 'flush' }]])
      expect(isFinished).toBe(false)
      queue.resolveFlush('a')
      await flushed

      expect(isFinished).toBe(true)
    })

    it('is resolved only for the note named', async () => {
      const { queue } = setup()
      let one = false
      let other = false
      const first = queue.flushPending('a').then(() => (one = true))
      queue
        .flushPending('b')
        .then(() => (other = true))
        .catch(rethrow)

      queue.resolveFlush('a')
      await first

      expect(one).toBe(true)
      expect(other).toBe(false)
    })

    it('is no trouble to resolve where nobody waits', () => {
      const { queue } = setup()

      expect(() => queue.resolveFlush('a')).not.toThrow()
    })
  })

  describe('forgetting a note', () => {
    it('stops its timer and resolves its flush', async () => {
      const { queue, turned } = setup()
      queue.startTimer('a', 100)
      let isFinished = false
      const flushed = queue.flushPending('a').then(() => (isFinished = true))

      queue.discard('a')
      await flushed
      vi.advanceTimersByTime(500)

      expect(isFinished).toBe(true)
      expect(turned).toEqual([['a', { kind: 'flush' }]])
    })
  })

  describe('a read', () => {
    it('passes the body and the file it came from to the tab', async () => {
      const { queue, turned } = setup({
        read: async () => asValue({ body: 'text', at: 'v1' }),
      })

      await queue.read('a', 'Note.md', 3)

      expect(turned).toEqual([
        ['a', { kind: 'read', generation: 3, answer: { kind: 'body', body: 'text', at: 'v1' } }],
      ])
    })

    it('passes an empty file mark where the note has none', async () => {
      const { queue, turned } = setup({ read: async () => asValue({ body: 'text' }) })

      await queue.read('a', 'Note.md', 1)

      expect(turned).toEqual([
        ['a', { kind: 'read', generation: 1, answer: { kind: 'body', body: 'text', at: '' } }],
      ])
    })

    it('reports a note that is not there as missing', async () => {
      const { queue, turned } = setup({ read: async () => asFailure('missing') })

      await queue.read('a', 'Note.md', 2)

      expect(turned).toEqual([['a', { kind: 'read', generation: 2, answer: { kind: 'missing' } }]])
    })

    it('reports any other failure by the name the tab knows it by', async () => {
      const { queue, turned } = setup({ read: async () => asFailure('alreadyExists') })

      await queue.read('a', 'Note.md', 2)

      expect(turned).toEqual([
        ['a', { kind: 'read', generation: 2, answer: { kind: 'error', error: 'unreadable' } }],
      ])
    })

    it('reports a call that threw as unreachable', async () => {
      const { queue, turned } = setup({
        read: async () => {
          throw new Error('down')
        },
      })

      await queue.read('a', 'Note.md', 4)

      expect(turned).toEqual([
        ['a', { kind: 'read', generation: 4, answer: { kind: 'error', error: 'unreachable' } }],
      ])
    })
  })

  describe('a write', () => {
    it('passes the file it wrote to the tab and lets a closing tab go on', async () => {
      const { queue, turned, resumed } = setup({
        write: async () => asValue({ body: '', at: 'v2' }),
      })

      await queue.write('a', { path: 'Note.md', body: 'text', baseline: null })

      expect(turned).toEqual([['a', { kind: 'written', answer: { kind: 'ok', at: 'v2' } }]])
      expect(resumed).toEqual(['a'])
    })

    it('passes an empty file mark where the write names none', async () => {
      const { queue, turned } = setup({ write: async () => asValue({ body: '' }) })

      await queue.write('a', { path: 'Note.md', body: 'text', baseline: null })

      expect(turned).toEqual([['a', { kind: 'written', answer: { kind: 'ok', at: '' } }]])
    })

    it('reports a file that changed as changed', async () => {
      const { queue, turned, resumed } = setup({ write: async () => asFailure('changed') })

      await queue.write('a', { path: 'Note.md', body: 'text', baseline: null })

      expect(turned).toEqual([['a', { kind: 'written', answer: { kind: 'changed' } }]])
      expect(resumed).toEqual(['a'])
    })

    it('reports any other failure by the name the tab knows it by', async () => {
      const { queue, turned } = setup({ write: async () => asFailure('invalidName') })

      await queue.write('a', { path: 'Note.md', body: 'text', baseline: null })

      expect(turned).toEqual([
        ['a', { kind: 'written', answer: { kind: 'error', error: 'unreadable' } }],
      ])
    })

    it('reports a call that threw as unreachable, and does not resume closing', async () => {
      const { queue, turned, resumed } = setup({
        write: async () => {
          throw new Error('down')
        },
      })

      await queue.write('a', { path: 'Note.md', body: 'text', baseline: null })

      expect(turned).toEqual([
        ['a', { kind: 'written', answer: { kind: 'error', error: 'unreachable' } }],
      ])
      expect(resumed).toEqual([])
    })
  })
})
