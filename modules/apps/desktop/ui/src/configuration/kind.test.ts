/**
 * The settings file held whole: read as it stands, written as it was typed, and
 * refused where the settings cannot be read out of it.
 *
 * Every value here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { holding, type Called } from './kind'
import { WORDS as words } from './words'

const HELD = '{\n  "agent": { "use": "claude" }\n}\n'

/** A vault holding that file, and everything it was asked to write. */
const standing = (answers: Partial<Called> = {}) => {
  const wrote: string[] = []
  const core: Called = {
    settingsFile: () => Promise.resolve({ written: HELD, path: '/numen.json' }),
    writesSettingsFile: (written) => {
      wrote.push(written)
      return Promise.resolve()
    },
    ...answers,
  }
  const reads = vi.fn()
  return { wrote, reads, held: holding(core, reads) }
}

describe('the file as it stands', () => {
  it('is read whole, byte for byte', async () => {
    const { held } = standing()
    await held.again()

    expect(held.text()).toBe(HELD)
    expect(held.read()).toBe(true)
    expect(held.changed()).toBe(false)
  })

  it('is nothing until it has been read', () => {
    const { held } = standing()
    expect(held.read()).toBe(false)
    expect(held.text()).toBe('')
  })

  it('says so where it could not be read', async () => {
    const { held } = standing({
      settingsFile: () => Promise.reject(new Error('the folder is not there')),
    })
    await held.again()

    expect(held.saying()).toBe(`${words.unread} the folder is not there`)
    expect(held.read()).toBe(false)
  })
})

describe('what is typed over it', () => {
  it('is marked as differing from what the file held', async () => {
    const { held } = standing()
    await held.again()

    held.types('{}\n')
    expect(held.changed()).toBe(true)

    held.types(HELD)
    expect(held.changed()).toBe(false)
  })

  it('is written as it was typed', async () => {
    const { held, wrote } = standing()
    await held.again()

    held.types('{\n  "agent": { "use": "" }\n}\n')
    await held.keeps()

    expect(wrote).toStrictEqual(['{\n  "agent": { "use": "" }\n}\n'])
    expect(held.changed()).toBe(false)
    expect(held.saying()).toBe('')
  })

  it('is written nowhere before the file has been read', async () => {
    const { held, wrote } = standing()
    held.types('{}\n')
    await held.keeps()

    expect(wrote).toStrictEqual([])
  })

  it('has every setting read again once it is written', async () => {
    const { held, reads } = standing()
    await held.again()
    held.types('{}\n')
    await held.keeps()

    expect(reads).toHaveBeenCalledTimes(1)
  })
})

describe('a file the settings cannot be read out of', () => {
  const refusing = () =>
    standing({
      writesSettingsFile: () =>
        Promise.reject(new Error('not a setting: it does not read as JSON, at byte 12')),
    })

  it('is refused, with what is wrong said', async () => {
    const { held } = refusing()
    await held.again()
    held.types('{ "agent": ')
    await held.keeps()

    expect(held.saying()).toContain(words.unwritten)
    expect(held.saying()).toContain('at byte 12')
  })

  it('is left in the editor, as it was typed', async () => {
    const { held } = refusing()
    await held.again()
    held.types('{ "agent": ')
    await held.keeps()

    expect(held.text()).toBe('{ "agent": ')
    expect(held.changed()).toBe(true)
  })

  it('has nothing read again', async () => {
    const { held, reads } = refusing()
    await held.again()
    held.types('{ "agent": ')
    await held.keeps()

    expect(reads).not.toHaveBeenCalled()
  })
})
