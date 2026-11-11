/** A fake vault and window for the tests of the preset writes: replies on demand, and records each message. */
import { afterEach, expect } from 'vitest'
import { formatErrorMessage, type MessageWriter, type Caught } from '@numen/plugin-sdk'
import { DEFAULTS, type Presets, type PresetWriteResult, type Settings } from '../../lib/presets'

const PATH = 'presets/Daily.md'

/** A write the test answers when it chooses. */
export const createGate = () => {
  let open: (answer: PresetWriteResult) => void = () => undefined
  const reply = new Promise<PresetWriteResult>((resolve) => {
    open = resolve
  })
  return { promise: reply, open }
}

const rejectUnused = (): never => {
  throw new Error('unused vault call')
}

/** A vault whose writes answer in turn, recording the version each was based on. */
export const createCore = (...answers: (PresetWriteResult | Promise<PresetWriteResult>)[]) => {
  const based: string[] = []
  const core: Presets = {
    read: rejectUnused,
    list: rejectUnused,
    createPreset: rejectUnused,
    scheduleDeck: rejectUnused,
    getDeckPreset: rejectUnused,
    curve: rejectUnused,
    write: async (_path: string, _settings: Settings, target: string) => {
      based.push(target)
      return (await answers.shift()) ?? { ok: true, value: { at: 'v-last' } }
    },
  }
  return { core, based }
}

/** The messages the window received, in order. */
export const createMessages = () => {
  const response: [string, string | undefined][] = []
  return {
    messages: response,
    said: response,
    writeMessage: (text: string, kind?: string) => {
      response.push([text, kind])
    },
  }
}

export const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

/** What a write of the defaults to PATH is sent with. */
export const createWriteOpts = (core: Presets, writeMessage: MessageWriter = () => undefined) => ({
  path: PATH,
  settings: DEFAULTS,
  core,
  writeMessage,
})

/**
 * A caller of `request` that does not wait for the write; a failure is kept, and each test ends by
 * expecting none.
 */
export const createBackgroundRequest = <Args extends object[]>(
  request: (...args: Args) => Promise<void>,
) => {
  const writeFailures: string[] = []
  afterEach(() => {
    expect(writeFailures).toStrictEqual([])
    writeFailures.length = 0
  })
  return (...args: Args): void => {
    request(...args).catch((error: Caught) => {
      writeFailures.push(formatErrorMessage(error))
    })
  }
}
