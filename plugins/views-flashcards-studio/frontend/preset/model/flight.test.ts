/**
 * A preset is written one write at a time: a write requested while another is in
 * progress runs after it, and a file changed externally is not overwritten.
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
import { canCloseTab, createWriteFlight, flushWrites, requestWrite, sendWrite } from './flight'

const requestInBackground = createBackgroundRequest(requestWrite)

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

    await sendWrite(flight, createWriteOpts(core, writeMessage))
    await sendWrite(flight, createWriteOpts(core, writeMessage))

    expect(based).toStrictEqual(['', 'v1'])
    expect(messages[0]).toStrictEqual(['', undefined])
  })

  it('takes back an earlier failure and warning once it lands', async () => {
    const flight = createWriteFlight()
    flight.errorMessage.value = words.unwritten
    flight.isWarned = true
    flight.theirs.add('retention')
    const { core } = createCore({ ok: true, value: { at: 'v1' } })

    await sendWrite(flight, createWriteOpts(core, createMessages().writeMessage))

    expect(flight.errorMessage.value).toBe('')
    expect(flight.isWarned).toBe(false)
    expect(flight.theirs.size).toBe(0)
  })

  it('says an error as an error', async () => {
    const flight = createWriteFlight()
    const { core } = createCore({ ok: false, error: 'notFound' } as PresetWriteResult)
    const { messages, writeMessage } = createMessages()

    await sendWrite(flight, createWriteOpts(core, writeMessage))

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

    await sendWrite(flight, createWriteOpts(core, createMessages().writeMessage))

    expect(flight.errorMessage.value).toBe(words.unwritten)
  })
})

describe('a write requested for while another is out', () => {
  it('goes after it', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core, based } = createCore(first.promise, { ok: true, value: { at: 'v2' } })
    const { writeMessage } = createMessages()

    const one = requestWrite(flight, createWriteOpts(core, writeMessage))
    const two = requestWrite(flight, createWriteOpts(core, writeMessage))
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

    const one = requestWrite(flight, createWriteOpts(core, writeMessage))
    let isSettled = false
    const two = requestWrite(flight, createWriteOpts(core, writeMessage)).then(() => {
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

    const one = requestWrite(flight, createWriteOpts(core, writeMessage))
    requestInBackground(flight, createWriteOpts(core, writeMessage))
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

    await flushWrites(flight, createWriteOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([''])
  })

  it('writes nothing where the person changed nothing', async () => {
    const flight = createWriteFlight()
    const { core, based } = createCore()

    await flushWrites(flight, createWriteOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([])
  })

  it('writes nothing over a file changed elsewhere', async () => {
    const flight = createWriteFlight()
    flight.theirs.add('retention')
    flight.hasChanged.value = true
    const { core, based } = createCore()

    await flushWrites(flight, createWriteOpts(core, createMessages().writeMessage))

    expect(based).toStrictEqual([])
  })

  it('waits for the write already out', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core } = createCore(first.promise)
    const { writeMessage } = createMessages()
    requestInBackground(flight, createWriteOpts(core, writeMessage))

    let isFlushed = false
    const flushed = flushWrites(flight, createWriteOpts(core, writeMessage)).then(() => {
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

    const isClosable = await canCloseTab(flight, createWriteOpts(core, writeMessage))

    expect(isClosable).toBe(true)
    expect(based).toStrictEqual([])
    expect(flight.isWarned).toBe(false)
  })

  it('writes what the person changed and closes', async () => {
    const flight = createWriteFlight()
    flight.theirs.add('retention')
    const { core, based } = createCore()

    const { writeMessage } = createMessages()

    const isClosable = await canCloseTab(flight, createWriteOpts(core, writeMessage))

    expect(isClosable).toBe(true)
    expect(based).toStrictEqual([''])
  })

  it('refuses once where the changes cannot be written, then closes', async () => {
    const flight = createWriteFlight()
    flight.theirs.add('retention')
    const denied = { ok: false, error: 'unreadable' } as unknown as PresetWriteResult
    const { core } = createCore(denied, denied)
    const { writeMessage } = createMessages()

    expect(await canCloseTab(flight, createWriteOpts(core, writeMessage))).toBe(false)
    expect(flight.isWarned).toBe(true)
    expect(await canCloseTab(flight, createWriteOpts(core, writeMessage))).toBe(true)
  })

  it('refuses where the file changed elsewhere', async () => {
    const flight = createWriteFlight()
    flight.hasChanged.value = true
    const { core, based } = createCore()

    const { writeMessage } = createMessages()

    const isClosable = await canCloseTab(flight, createWriteOpts(core, writeMessage))

    expect(isClosable).toBe(false)
    expect(based).toStrictEqual([])
  })
})
