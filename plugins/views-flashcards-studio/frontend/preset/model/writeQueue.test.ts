/**
 * A preset is written one write at a time: a write requested while another is
 * in flight is queued after it, and a file changed externally is not overwritten.
 */
import { describe, expect, it } from 'vitest'
import {
  createBackgroundRequest,
  createCore,
  createGate,
  createMessages,
  createWriteOpts,
  flushPromises,
} from '../fixtures/writes'
import type { Presets, PresetWriteResult } from '../../lib/presets'
import { WORDS as words } from '../lib/words'
import { createWriteQueue, flushWrites, requestWrite, sendWrite } from './writeQueue'

const requestInBackground = createBackgroundRequest(requestWrite)

describe('a preset nothing has been written to yet', () => {
  it('remains at no version, with nothing queued, changed or returned', () => {
    const queue = createWriteQueue()

    expect(queue.unsaved.revision).toBe('')
    expect(queue.inflight.isWriting).toBe(false)
    expect(queue.inflight.isWriteQueued).toBe(false)
    expect(queue.inflight.activeWrite).toBeNull()
    expect(queue.unsaved.hasWarned).toBe(false)
    expect(queue.unsaved.dirtyFields.size).toBe(0)
    expect(queue.status.hasExternalChange.value).toBe(false)
    expect(queue.status.errorMessage.value).toBe('')
  })
})

describe('one write', () => {
  it('clears the transcript, and is based on the version last written', async () => {
    const queue = createWriteQueue()
    const { core, based } = createCore({ ok: true, value: { at: 'v1' } })
    const { messages, writeMessage } = createMessages()

    await sendWrite(queue, createWriteOpts(core, writeMessage))
    await sendWrite(queue, createWriteOpts(core, writeMessage))

    expect(based).toStrictEqual(['', 'v1'])
    expect(messages[0]).toStrictEqual(['', undefined])
  })

  it('takes back an earlier failure and warning once it lands', async () => {
    const queue = createWriteQueue()
    queue.status.errorMessage.value = words.unwritten
    queue.unsaved.hasWarned = true
    queue.unsaved.dirtyFields.add('retention')
    const { core } = createCore({ ok: true, value: { at: 'v1' } })

    await sendWrite(queue, createWriteOpts(core, createMessages().writeMessage))

    expect(queue.status.errorMessage.value).toBe('')
    expect(queue.unsaved.hasWarned).toBe(false)
    expect(queue.unsaved.dirtyFields.size).toBe(0)
  })

  it('says an error as an error', async () => {
    const queue = createWriteQueue()
    const { core } = createCore({ ok: false, error: 'notFound' } as PresetWriteResult)
    const { messages, writeMessage } = createMessages()

    await sendWrite(queue, createWriteOpts(core, writeMessage))

    expect(messages.at(-1)).toStrictEqual([queue.status.errorMessage.value, 'error'])
  })
})

describe('a write the vault could not be requested for', () => {
  it('says the settings were not written', async () => {
    const queue = createWriteQueue()
    const core = {
      write: async () => {
        throw new Error('connection failed')
      },
    } as unknown as Presets

    await sendWrite(queue, createWriteOpts(core, createMessages().writeMessage))

    expect(queue.status.errorMessage.value).toBe(words.unwritten)
  })
})

describe('a write requested for while another is out', () => {
  it('goes after it', async () => {
    const queue = createWriteQueue()
    const first = createGate()
    const { core, based } = createCore(first.promise, { ok: true, value: { at: 'v2' } })
    const { writeMessage } = createMessages()

    const one = requestWrite(queue, createWriteOpts(core, writeMessage))
    const two = requestWrite(queue, createWriteOpts(core, writeMessage))
    expect(queue.inflight.isWriteQueued).toBe(true)
    first.open({ ok: true, value: { at: 'v1' } })
    await Promise.all([one, two])

    expect(based).toStrictEqual(['', 'v1'])
    expect(queue.unsaved.revision).toBe('v2')
    expect(queue.inflight.activeWrite).toBeNull()
  })

  it('is dropped where the first found the file changed elsewhere', async () => {
    const queue = createWriteQueue()
    const first = createGate()
    const { core, based } = createCore(first.promise)
    const { writeMessage } = createMessages()

    const one = requestWrite(queue, createWriteOpts(core, writeMessage))
    requestInBackground(queue, createWriteOpts(core, writeMessage))
    first.open({ ok: false, error: 'changed' } as PresetWriteResult)
    await one

    expect(based).toStrictEqual([''])
    expect(queue.status.hasExternalChange.value).toBe(true)
  })
})

describe('flushing a preset', () => {
  it('writes what the person changed', async () => {
    const queue = createWriteQueue()
    queue.unsaved.dirtyFields.add('retention')
    const { core, based } = createCore()

    await flushWrites(queue, createWriteOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([''])
  })

  it('writes nothing where the person changed nothing', async () => {
    const queue = createWriteQueue()
    const { core, based } = createCore()

    await flushWrites(queue, createWriteOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([])
  })

  it('writes nothing over a file changed elsewhere', async () => {
    const queue = createWriteQueue()
    queue.unsaved.dirtyFields.add('retention')
    queue.status.hasExternalChange.value = true
    const { core, based } = createCore()

    await flushWrites(queue, createWriteOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([])
  })

  it('waits for the write already out', async () => {
    const queue = createWriteQueue()
    const first = createGate()
    const { core } = createCore(first.promise)
    const { writeMessage } = createMessages()
    requestInBackground(queue, createWriteOpts(core, writeMessage))

    let isFlushed = false
    const flushed = flushWrites(queue, createWriteOpts(core, writeMessage)).then(() => {
      isFlushed = true
    })
    await flushPromises()
    expect(isFlushed).toBe(false)

    first.open({ ok: true, value: { at: 'v1' } })
    await flushed
    expect(isFlushed).toBe(true)
  })
})
