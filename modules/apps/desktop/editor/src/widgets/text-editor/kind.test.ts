/**
 * The settings file held whole: read as it stands, written as it was typed, and
 * refused where the settings cannot be read out of it.
 *
 * Every value here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { useSettingsFileTab, type SettingsFileTabDeps } from './kind'
import { WORDS as words } from './words'

const HELD = '{\n  "agent": { "use": "claude" }\n}\n'

/** A vault holding that file, and everything it was asked to write. */
const vault = (answers: Partial<SettingsFileTabDeps> = {}) => {
  const wrote: string[] = []
  /** What each write presented as the file it last read. */
  const presented: (string | null)[] = []
  const core: SettingsFileTabDeps = {
    settingsFile: () => Promise.resolve({ written: HELD, path: '/numen.json' }),
    writesSettingsFile: (written, seen) => {
      wrote.push(written)
      presented.push(seen)
      return Promise.resolve({ changed: false })
    },
    ...answers,
  }
  const reads = vi.fn()
  return { wrote, presented, reads, held: useSettingsFileTab(core, reads) }
}

describe('the file as it stands', () => {
  it('is read whole, byte for byte', async () => {
    const { held } = vault()
    await held.again()

    expect(held.text.value).toBe(HELD)
    expect(held.read.value).toBe(true)
    expect(held.changed.value).toBe(false)
  })

  it('is nothing until it has been read', () => {
    const { held } = vault()
    expect(held.read.value).toBe(false)
    expect(held.text.value).toBe('')
  })

  it('says so where it could not be read', async () => {
    const { held } = vault({
      settingsFile: () => Promise.reject(new Error('the folder is not there')),
    })
    await held.again()

    expect(held.errorMessage.value).toBe(
      `${words.unread} numen did not answer, so nothing was done — it may have stopped, and the window keeps trying`,
    )
    expect(held.read.value).toBe(false)
  })
})

describe('what is typed over it', () => {
  it('is marked as differing from what the file held', async () => {
    const { held } = vault()
    await held.again()

    held.types('{}\n')
    expect(held.changed.value).toBe(true)

    held.types(HELD)
    expect(held.changed.value).toBe(false)
  })

  it('is written as it was typed', async () => {
    const { held, wrote } = vault()
    await held.again()

    held.types('{\n  "agent": { "use": "" }\n}\n')
    await held.keeps()

    expect(wrote).toStrictEqual(['{\n  "agent": { "use": "" }\n}\n'])
    expect(held.changed.value).toBe(false)
    expect(held.errorMessage.value).toBe('')
  })

  it('presents the file the tab last read', async () => {
    const { held, presented } = vault()
    await held.again()

    held.types('{}\n')
    await held.keeps()

    expect(presented).toStrictEqual([HELD])
  })

  it('is written nowhere before the file has been read', async () => {
    const { held, wrote } = vault()
    held.types('{}\n')
    await held.keeps()

    expect(wrote).toStrictEqual([])
  })

  it('has every setting read again once it is written', async () => {
    const { held, reads } = vault()
    await held.again()
    held.types('{}\n')
    await held.keeps()

    expect(reads).toHaveBeenCalledTimes(1)
  })
})

describe('a file the settings cannot be read out of', () => {
  const refusing = () =>
    vault({
      writesSettingsFile: () =>
        Promise.reject(
          new ConnectError('not a setting: it does not read as JSON, at byte 12', Code.InvalidArgument),
        ),
    })

  it('is refused, with what is wrong said', async () => {
    const { held } = refusing()
    await held.again()
    held.types('{ "agent": ')
    await held.keeps()

    expect(held.errorMessage.value).toContain(words.unwritten)
    expect(held.errorMessage.value).toContain('at byte 12')
  })

  it('is left in the editor, as it was typed', async () => {
    const { held } = refusing()
    await held.again()
    held.types('{ "agent": ')
    await held.keeps()

    expect(held.text.value).toBe('{ "agent": ')
    expect(held.changed.value).toBe(true)
  })

  it('has nothing read again', async () => {
    const { held, reads } = refusing()
    await held.again()
    held.types('{ "agent": ')
    await held.keeps()

    expect(reads).not.toHaveBeenCalled()
  })
})

describe('a file that moved past what the tab read', () => {
  const MOVED = '{\n  "agent": { "use": "claude" },\n  "review": { "day_starts": "05:00" }\n}\n'
  const TYPED = '{\n  "agent": { "use": "gemini" }\n}\n'

  /**
   * The tab reads the file, the settings page patches it, and the tab keeps
   * what it has. A write presenting anything but the file as it stands is
   * refused.
   */
  const stale = async () => {
    let stands = HELD
    const wrote: string[] = []
    const reads = vi.fn()
    const core: SettingsFileTabDeps = {
      settingsFile: () => Promise.resolve({ written: stands, path: '/numen.json' }),
      writesSettingsFile: (written, seen) => {
        if (seen !== null && seen !== stands) return Promise.resolve({ changed: true })
        stands = written
        wrote.push(written)
        return Promise.resolve({ changed: false })
      },
    }
    const held = useSettingsFileTab(core, reads)
    await held.again()
    stands = MOVED
    held.types(TYPED)
    await held.keeps()
    return { held, wrote, reads }
  }

  it('stops keeping, with nothing written', async () => {
    const { held, wrote, reads } = await stale()

    expect(held.isStale.value).toBe(true)
    expect(wrote).toStrictEqual([])
    expect(reads).not.toHaveBeenCalled()
  })

  it('writes what was typed where the person keeps theirs', async () => {
    const { held, wrote, reads } = await stale()
    await held.keep()

    expect(wrote).toStrictEqual([TYPED])
    expect(held.isStale.value).toBe(false)
    expect(held.changed.value).toBe(false)
    expect(reads).toHaveBeenCalledTimes(1)
  })

  it("reads the file again where the person takes the file's", async () => {
    const { held, wrote } = await stale()
    await held.take()

    expect(held.text.value).toBe(MOVED)
    expect(held.isStale.value).toBe(false)
    expect(held.changed.value).toBe(false)
    expect(wrote).toStrictEqual([])
  })

  it('is written nowhere until the person answers', async () => {
    const { held, wrote } = await stale()
    await held.keeps()

    expect(held.isStale.value).toBe(true)
    expect(wrote).toStrictEqual([])
  })
})
