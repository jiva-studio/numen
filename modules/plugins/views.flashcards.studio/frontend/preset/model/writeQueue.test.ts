/**
 * A preset is written one write at a time: a write asked for while another is
 * out goes after it, and a file changed elsewhere is not written over.
 */
import { describe, expect, it } from 'vitest'
import type { MessageWriter } from '@numen/plugin-sdk'
import { DEFAULTS, type Presets, type PresetWriteResult } from '../types'
import { WORDS as words } from '../words'
import { createWriteQueue, flushWrites, requestWrite, sendWrite } from './writeQueue'

const PATH = 'presets/Daily.md'

/** A write the test answers when it chooses. */
const createGate = () => {
  let open: (answer: PresetWriteResult) => void = () => {}
  const reply = new Promise<PresetWriteResult>((resolve) => {
    open = resolve
  })
  return { promise: reply, open }
}

/** A vault whose writes answer in turn, recording the version each was based on. */
const createCore = (...answers: (PresetWriteResult | Promise<PresetWriteResult>)[]) => {
  const based: string[] = []
  const core = {
    write: async (_path: string, _settings: unknown, target: string) => {
      based.push(target)
      return (await answers.shift()) ?? { ok: true, value: { at: 'v-last' } }
    },
  } as unknown as Presets
  return { core, based }
}

/** What the window was told, in order. */
const createMessages = () => {
  const response: [string, string | undefined][] = []
  return {
    messages: response,
    said: response,
    writeMessage: (text: string, kind?: string) => void response.push([text, kind]),
  }
}

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

const createWriteOpts = (core: Presets, writeMessage: MessageWriter = () => {}) => ({
  path: PATH,
  settings: DEFAULTS,
  core,
  writeMessage,
})

describe('a preset nothing has been written to yet', () => {
  it('remains at no version, with nothing queued, changed or returned', () => {
    const queue = createWriteQueue()

    expect(queue.revision).toBe('')
    expect(queue.isWriting).toBe(false)
    expect(queue.isWriteQueued).toBe(false)
    expect(queue.activeWrite).toBeNull()
    expect(queue.hasWarned).toBe(false)
    expect(queue.dirtyFields.size).toBe(0)
    expect(queue.hasExternalChange.value).toBe(false)
    expect(queue.errorMessage.value).toBe('')
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
    queue.errorMessage.value = words.unwritten
    queue.hasWarned = true
    queue.dirtyFields.add('retention')
    const { core } = createCore({ ok: true, value: { at: 'v1' } })

    await sendWrite(queue, createWriteOpts(core, createMessages().writeMessage))

    expect(queue.errorMessage.value).toBe('')
    expect(queue.hasWarned).toBe(false)
    expect(queue.dirtyFields.size).toBe(0)
  })

  it('says an error as an error', async () => {
    const queue = createWriteQueue()
    const { core } = createCore({ ok: false, error: 'missing' } as PresetWriteResult)
    const { messages, writeMessage } = createMessages()

    await sendWrite(queue, createWriteOpts(core, writeMessage))

    expect(messages.at(-1)).toStrictEqual([queue.errorMessage.value, 'error'])
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

    expect(queue.errorMessage.value).toBe(words.unwritten)
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
    expect(queue.isWriteQueued).toBe(true)
    first.open({ ok: true, value: { at: 'v1' } })
    await Promise.all([one, two])

    expect(based).toStrictEqual(['', 'v1'])
    expect(queue.revision).toBe('v2')
    expect(queue.activeWrite).toBeNull()
  })

  it('is dropped where the first found the file changed elsewhere', async () => {
    const queue = createWriteQueue()
    const first = createGate()
    const { core, based } = createCore(first.promise)
    const { writeMessage } = createMessages()

    const one = requestWrite(queue, createWriteOpts(core, writeMessage))
    void requestWrite(queue, createWriteOpts(core, writeMessage))
    first.open({ ok: false, error: 'changed' } as PresetWriteResult)
    await one

    expect(based).toStrictEqual([''])
    expect(queue.hasExternalChange.value).toBe(true)
  })
})

describe('flushing a preset', () => {
  it('writes what the person changed', async () => {
    const queue = createWriteQueue()
    queue.dirtyFields.add('retention')
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
    queue.dirtyFields.add('retention')
    queue.hasExternalChange.value = true
    const { core, based } = createCore()

    await flushWrites(queue, createWriteOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([])
  })

  it('waits for the write already out', async () => {
    const queue = createWriteQueue()
    const first = createGate()
    const { core } = createCore(first.promise)
    const { writeMessage } = createMessages()
    void requestWrite(queue, createWriteOpts(core, writeMessage))

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
