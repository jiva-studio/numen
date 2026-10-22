/**
 * A preset is written one write at a time: a write asked for while another is
 * out goes after it, and a file changed elsewhere is not written over.
 */
import { describe, expect, it } from 'vitest'
import type { MessageWriter } from '@numen/plugin-sdk'
import { DEFAULTS, type Presets, type PresetWriteResult } from '../types'
import { WORDS as words } from '../words'
import { canCloseTab, createWriteFlight, flushWrites, requestWrite, sendWrite } from './flight'

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

const createFlightOpts = (core: Presets, writeMessage: MessageWriter = () => {}) => ({
  path: PATH,
  settings: DEFAULTS,
  core,
  writeMessage,
})

describe('a preset nothing has been written to yet', () => {
  it('remains at no version, with nothing queued, changed or returned', () => {
    const flight = createWriteFlight()

    expect(flight.at).toBe('')
    expect(flight.isWriting).toBe(false)
    expect(flight.isWriteQueued).toBe(false)
    expect(flight.flight).toBeNull()
    expect(flight.isWarned).toBe(false)
    expect(flight.theirs.size).toBe(0)
    expect(flight.hasChanged.value).toBe(false)
    expect(flight.errorMessage.value).toBe('')
  })
})

describe('one write', () => {
  it('clears the transcript, and is based on the version last written', async () => {
    const flight = createWriteFlight()
    const { core, based } = createCore({ ok: true, value: { at: 'v1' } })
    const { messages, writeMessage } = createMessages()

    await sendWrite(flight, createFlightOpts(core, writeMessage))
    await sendWrite(flight, createFlightOpts(core, writeMessage))

    expect(based).toStrictEqual(['', 'v1'])
    expect(messages[0]).toStrictEqual(['', undefined])
  })

  it('takes back an earlier failure and warning once it lands', async () => {
    const flight = createWriteFlight()
    flight.errorMessage.value = words.unwritten
    flight.isWarned = true
    flight.theirs.add('retention')
    const { core } = createCore({ ok: true, value: { at: 'v1' } })

    await sendWrite(flight, createFlightOpts(core, createMessages().writeMessage))

    expect(flight.errorMessage.value).toBe('')
    expect(flight.isWarned).toBe(false)
    expect(flight.theirs.size).toBe(0)
  })

  it('says an error as an error', async () => {
    const flight = createWriteFlight()
    const { core } = createCore({ ok: false, error: 'missing' } as PresetWriteResult)
    const { messages, writeMessage } = createMessages()

    await sendWrite(flight, createFlightOpts(core, writeMessage))

    expect(messages.at(-1)).toStrictEqual([flight.errorMessage.value, 'error'])
  })
})

describe('a write the vault could not be requested for', () => {
  it('says the settings were not written', async () => {
    const flight = createWriteFlight()
    const core = {
      write: async () => {
        throw new Error('connection failed')
      },
    } as unknown as Presets

    await sendWrite(flight, createFlightOpts(core, createMessages().writeMessage))

    expect(flight.errorMessage.value).toBe(words.unwritten)
  })
})

describe('a write requested for while another is out', () => {
  it('goes after it', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core, based } = createCore(first.promise, { ok: true, value: { at: 'v2' } })
    const { writeMessage } = createMessages()

    const one = requestWrite(flight, createFlightOpts(core, writeMessage))
    const two = requestWrite(flight, createFlightOpts(core, writeMessage))
    expect(flight.isWriteQueued).toBe(true)
    first.open({ ok: true, value: { at: 'v1' } })
    await Promise.all([one, two])

    expect(based).toStrictEqual(['', 'v1'])
    expect(flight.at).toBe('v2')
    expect(flight.flight).toBeNull()
  })

  it('is not settled until the write out has landed', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core } = createCore(first.promise)
    const { writeMessage } = createMessages()

    const one = requestWrite(flight, createFlightOpts(core, writeMessage))
    let isSettled = false
    const two = requestWrite(flight, createFlightOpts(core, writeMessage)).then(() => {
      isSettled = true
    })
    await flushPromises()
    expect(isSettled).toBe(false)

    first.open({ ok: true, value: { at: 'v1' } })
    await Promise.all([one, two])
    expect(isSettled).toBe(true)
  })

  it('is dropped where the first found the file changed elsewhere', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core, based } = createCore(first.promise)
    const { writeMessage } = createMessages()

    const one = requestWrite(flight, createFlightOpts(core, writeMessage))
    void requestWrite(flight, createFlightOpts(core, writeMessage))
    first.open({ ok: false, error: 'changed' } as PresetWriteResult)
    await one

    expect(based).toStrictEqual([''])
    expect(flight.hasChanged.value).toBe(true)
  })
})

describe('flushing a preset', () => {
  it('writes what the person changed', async () => {
    const flight = createWriteFlight()
    flight.theirs.add('retention')
    const { core, based } = createCore()

    await flushWrites(flight, createFlightOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([''])
  })

  it('writes nothing where the person changed nothing', async () => {
    const flight = createWriteFlight()
    const { core, based } = createCore()

    await flushWrites(flight, createFlightOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([])
  })

  it('writes nothing over a file changed elsewhere', async () => {
    const flight = createWriteFlight()
    flight.theirs.add('retention')
    flight.hasChanged.value = true
    const { core, based } = createCore()

    await flushWrites(flight, createFlightOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([])
  })

  it('waits for the write already out', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core } = createCore(first.promise)
    const { writeMessage } = createMessages()
    void requestWrite(flight, createFlightOpts(core, writeMessage))

    let isFlushed = false
    const flushed = flushWrites(flight, createFlightOpts(core, writeMessage)).then(() => {
      isFlushed = true
    })
    await flushPromises()
    expect(isFlushed).toBe(false)

    first.open({ ok: true, value: { at: 'v1' } })
    await flushed
    expect(isFlushed).toBe(true)
  })
})

describe('closing a preset tab', () => {
  it('closes at once where nothing is unwritten', async () => {
    const flight = createWriteFlight()
    const { core, based } = createCore()

    const { writeMessage } = createMessages()

    const isClosable = await canCloseTab(flight, createFlightOpts(core, writeMessage))

    expect(isClosable).toBe(true)
    expect(based).toStrictEqual([])
    expect(flight.isWarned).toBe(false)
  })

  it('writes what the person changed and closes', async () => {
    const flight = createWriteFlight()
    flight.theirs.add('retention')
    const { core, based } = createCore()

    const { writeMessage } = createMessages()

    const isClosable = await canCloseTab(flight, createFlightOpts(core, writeMessage))

    expect(isClosable).toBe(true)
    expect(based).toStrictEqual([''])
  })

  it('refuses once where the changes cannot be written, then closes', async () => {
    const flight = createWriteFlight()
    flight.theirs.add('retention')
    const denied = { ok: false, error: 'denied' } as unknown as PresetWriteResult
    const { core } = createCore(denied, denied)
    const { writeMessage } = createMessages()

    expect(await canCloseTab(flight, createFlightOpts(core, writeMessage))).toBe(false)
    expect(flight.isWarned).toBe(true)
    expect(await canCloseTab(flight, createFlightOpts(core, writeMessage))).toBe(true)
  })

  it('refuses where the file changed elsewhere', async () => {
    const flight = createWriteFlight()
    flight.hasChanged.value = true
    const { core, based } = createCore()

    const { writeMessage } = createMessages()

    const isClosable = await canCloseTab(flight, createFlightOpts(core, writeMessage))

    expect(isClosable).toBe(false)
    expect(based).toStrictEqual([])
  })
})
