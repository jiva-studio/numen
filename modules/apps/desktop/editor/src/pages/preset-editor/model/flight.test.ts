/**
 * A preset is written one write at a time: a write asked for while another is
 * out goes after it, and a file changed elsewhere is not written over.
 */
import { describe, expect, it } from 'vitest'
import { DEFAULTS, type Presets, type PresetWriteResult } from '../types'
import { WORDS as words } from '../words'
import { createWriteFlight, flushWrites, requestWrite, sendWrite } from './flight'

const PATH = 'presets/Daily.md'

/** A write the test answers when it chooses. */
const createGate = () => {
  let open: (answer: PresetWriteResult) => void = () => {}
  const answered = new Promise<PresetWriteResult>((resolve) => {
    open = resolve
  })
  return { answered, open }
}

/** A vault whose writes answer in turn, recording the version each was based on. */
const createCore = (...answers: (PresetWriteResult | Promise<PresetWriteResult>)[]) => {
  const based: string[] = []
  const core = {
    write: async (_path: string, _settings: unknown, at: string) => {
      based.push(at)
      return (await answers.shift()) ?? { ok: true, value: { at: 'v-last' } }
    },
  } as unknown as Presets
  return { core, based }
}

/** What the window was told, in order. */
const createMessages = () => {
  const said: [string, string | undefined][] = []
  return { said, writeMessage: (text: string, kind?: string) => void said.push([text, kind]) }
}

const nap = () => new Promise((wake) => setTimeout(wake, 0))

describe('a preset nothing has been written to yet', () => {
  it('stands at no version, with nothing queued, changed or said', () => {
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
  it('clears what was said, and is based on the version last written', async () => {
    const flight = createWriteFlight()
    const { core, based } = createCore({ ok: true, value: { at: 'v1' } })
    const { said, writeMessage } = createMessages()

    await sendWrite(flight, PATH, DEFAULTS, core, writeMessage)
    await sendWrite(flight, PATH, DEFAULTS, core, writeMessage)

    expect(based).toStrictEqual(['', 'v1'])
    expect(said[0]).toStrictEqual(['', undefined])
  })

  it('takes back an earlier failure and warning once it lands', async () => {
    const flight = createWriteFlight()
    flight.errorMessage.value = words.unwritten
    flight.isWarned = true
    flight.theirs.add('retention')
    const { core } = createCore({ ok: true, value: { at: 'v1' } })

    await sendWrite(flight, PATH, DEFAULTS, core, createMessages().writeMessage)

    expect(flight.errorMessage.value).toBe('')
    expect(flight.isWarned).toBe(false)
    expect(flight.theirs.size).toBe(0)
  })

  it('says a refusal as an error', async () => {
    const flight = createWriteFlight()
    const { core } = createCore({ ok: false, error: 'notFound' } as PresetWriteResult)
    const { said, writeMessage } = createMessages()

    await sendWrite(flight, PATH, DEFAULTS, core, writeMessage)

    expect(said.at(-1)).toStrictEqual([flight.errorMessage.value, 'error'])
  })
})

describe('a write asked for while another is out', () => {
  it('goes after it', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core, based } = createCore(first.answered, { ok: true, value: { at: 'v2' } })
    const { writeMessage } = createMessages()

    const one = requestWrite(flight, PATH, DEFAULTS, core, writeMessage)
    const two = requestWrite(flight, PATH, DEFAULTS, core, writeMessage)
    expect(flight.isWriteQueued).toBe(true)
    first.open({ ok: true, value: { at: 'v1' } })
    await Promise.all([one, two])

    expect(based).toStrictEqual(['', 'v1'])
    expect(flight.at).toBe('v2')
    expect(flight.flight).toBeNull()
  })

  it('is dropped where the first found the file changed elsewhere', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core, based } = createCore(first.answered)
    const { writeMessage } = createMessages()

    const one = requestWrite(flight, PATH, DEFAULTS, core, writeMessage)
    void requestWrite(flight, PATH, DEFAULTS, core, writeMessage)
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

    await flushWrites(flight, PATH, DEFAULTS, core, createMessages().writeMessage)

    expect(based).toStrictEqual([''])
  })

  it('writes nothing where the person changed nothing', async () => {
    const flight = createWriteFlight()
    const { core, based } = createCore()

    await flushWrites(flight, PATH, DEFAULTS, core, createMessages().writeMessage)

    expect(based).toStrictEqual([])
  })

  it('writes nothing over a file changed elsewhere', async () => {
    const flight = createWriteFlight()
    flight.theirs.add('retention')
    flight.hasChanged.value = true
    const { core, based } = createCore()

    await flushWrites(flight, PATH, DEFAULTS, core, createMessages().writeMessage)

    expect(based).toStrictEqual([])
  })

  it('waits for the write already out', async () => {
    const flight = createWriteFlight()
    const first = createGate()
    const { core } = createCore(first.answered)
    const { writeMessage } = createMessages()
    void requestWrite(flight, PATH, DEFAULTS, core, writeMessage)

    let isFlushed = false
    const flushed = flushWrites(flight, PATH, DEFAULTS, core, writeMessage).then(() => {
      isFlushed = true
    })
    await nap()
    expect(isFlushed).toBe(false)

    first.open({ ok: true, value: { at: 'v1' } })
    await flushed
    expect(isFlushed).toBe(true)
  })
})
