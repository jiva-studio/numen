/**
 * The settings file tab: reads the raw file, saves the draft unchanged, and reports
 * text that does not parse as settings.
 *
 * Every value here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { useSettingsFile, type SettingsFileTabDeps } from './useSettingsFile'
import { WORDS as words } from '../words'

const FILE_TEXT = '{\n  "agent": { "use": "claude" }\n}\n'

/** A vault with that file, recording every write. */
const vault = (overrides: Partial<SettingsFileTabDeps> = {}) => {
  const writes: string[] = []
  /** The expected version passed with each write. */
  const versions: (string | null)[] = []
  const core: SettingsFileTabDeps = {
    getSettingsFile: () => Promise.resolve({ written: FILE_TEXT, path: '/numen.json' }),
    saveSettingsFile: (content, expectedVersion) => {
      writes.push(content)
      versions.push(expectedVersion)
      return Promise.resolve({ isChanged: false })
    },
    ...overrides,
  }
  const reads = vi.fn()
  return { writes, versions, reads, tab: useSettingsFile(core, reads) }
}

describe('reading the file', () => {
  it('reads the file byte for byte', async () => {
    const { tab } = vault()
    await tab.reload()

    expect(tab.text.value).toBe(FILE_TEXT)
    expect(tab.isRead.value).toBe(true)
    expect(tab.changed.value).toBe(false)
  })

  it('is empty before the first read', () => {
    const { tab } = vault()
    expect(tab.isRead.value).toBe(false)
    expect(tab.text.value).toBe('')
    expect(tab.changed.value).toBe(false)
    expect(tab.errorMessage.value).toBe('')
    expect(tab.isStale.value).toBe(false)
  })

  it('clears the error after a successful read follows a failed one', async () => {
    let fails = true
    const { tab } = vault({
      getSettingsFile: () =>
        fails
          ? Promise.reject(new Error('the folder is not there'))
          : Promise.resolve({ written: FILE_TEXT, path: '/numen.json' }),
    })
    await tab.reload()
    expect(tab.errorMessage.value).not.toBe('')

    fails = false
    await tab.reload()

    expect(tab.errorMessage.value).toBe('')
  })

  it('reports a failed read', async () => {
    const { tab } = vault({
      getSettingsFile: () => Promise.reject(new Error('the folder is not there')),
    })
    await tab.reload()

    expect(tab.errorMessage.value).toBe(
      `${words.unread} numen did not answer, so nothing was done — it may have stopped, and the window keeps trying`,
    )
    expect(tab.isRead.value).toBe(false)
  })
})

describe('editing the draft', () => {
  it('marks the draft as changed while it differs from the file', async () => {
    const { tab } = vault()
    await tab.reload()

    tab.type('{}\n')
    expect(tab.changed.value).toBe(true)

    tab.type(FILE_TEXT)
    expect(tab.changed.value).toBe(false)
  })

  it('saves the draft unchanged', async () => {
    const { tab, writes } = vault()
    await tab.reload()

    tab.type('{\n  "agent": { "use": "" }\n}\n')
    await tab.save()

    expect(writes).toStrictEqual(['{\n  "agent": { "use": "" }\n}\n'])
    expect(tab.changed.value).toBe(false)
    expect(tab.errorMessage.value).toBe('')
  })

  it('passes the last read text as the expected version', async () => {
    const { tab, versions } = vault()
    await tab.reload()

    tab.type('{}\n')
    await tab.save()

    expect(versions).toStrictEqual([FILE_TEXT])
  })

  it('writes nothing before the first read', async () => {
    const { tab, writes } = vault()
    tab.type('{}\n')
    await tab.save()

    expect(writes).toStrictEqual([])
  })

  it('reloads the settings after a save', async () => {
    const { tab, reads } = vault()
    await tab.reload()
    tab.type('{}\n')
    await tab.save()

    expect(reads).toHaveBeenCalledTimes(1)
  })
})

describe('a draft that does not parse as settings', () => {
  const createUnwritableVault = () =>
    vault({
      saveSettingsFile: () =>
        Promise.reject(
          new ConnectError(
            'not a setting: it does not read as JSON, at byte 12',
            Code.InvalidArgument,
          ),
        ),
    })

  it('reports the parse error', async () => {
    const { tab } = createUnwritableVault()
    await tab.reload()
    tab.type('{ "agent": ')
    await tab.save()

    expect(tab.errorMessage.value).toContain(words.unwritten)
    expect(tab.errorMessage.value).toContain('at byte 12')
  })

  it('keeps the draft in the editor', async () => {
    const { tab } = createUnwritableVault()
    await tab.reload()
    tab.type('{ "agent": ')
    await tab.save()

    expect(tab.text.value).toBe('{ "agent": ')
    expect(tab.changed.value).toBe(true)
  })

  it('does not reload the settings', async () => {
    const { tab, reads } = createUnwritableVault()
    await tab.reload()
    tab.type('{ "agent": ')
    await tab.save()

    expect(reads).not.toHaveBeenCalled()
  })
})

describe('a file that changed on disk after the read', () => {
  const MOVED = '{\n  "agent": { "use": "claude" },\n  "review": { "day_start_time": "05:00" }\n}\n'
  const DRAFT = '{\n  "agent": { "use": "gemini" }\n}\n'

  const createStaleTab = async () => {
    let diskText = FILE_TEXT
    const writes: string[] = []
    const reads = vi.fn()
    const core: SettingsFileTabDeps = {
      getSettingsFile: () => Promise.resolve({ written: diskText, path: '/numen.json' }),
      saveSettingsFile: (content, expectedVersion) => {
        if (expectedVersion !== null && expectedVersion !== diskText) {
          return Promise.resolve({ isChanged: true })
        }
        diskText = content
        writes.push(content)
        return Promise.resolve({ isChanged: false })
      },
    }
    const tab = useSettingsFile(core, reads)
    await tab.reload()
    diskText = MOVED
    tab.type(DRAFT)
    await tab.save()
    return { tab, writes, reads }
  }

  it('marks the tab stale and writes nothing', async () => {
    const { tab, writes, reads } = await createStaleTab()

    expect(tab.isStale.value).toBe(true)
    expect(writes).toStrictEqual([])
    expect(reads).not.toHaveBeenCalled()
  })

  it('writes the draft on keep', async () => {
    const { tab, writes, reads } = await createStaleTab()
    await tab.keep()

    expect(writes).toStrictEqual([DRAFT])
    expect(tab.isStale.value).toBe(false)
    expect(tab.changed.value).toBe(false)
    expect(reads).toHaveBeenCalledTimes(1)
  })

  it('reloads the file on take', async () => {
    const { tab, writes } = await createStaleTab()
    await tab.take()

    expect(tab.text.value).toBe(MOVED)
    expect(tab.isStale.value).toBe(false)
    expect(tab.changed.value).toBe(false)
    expect(writes).toStrictEqual([])
  })

  it('ignores keep when the file is not stale', async () => {
    const { tab, writes } = vault()
    await tab.reload()
    tab.type('{}\n')
    await tab.keep()

    expect(writes).toStrictEqual([])
  })

  it('does not save until keep or take', async () => {
    const { tab, writes } = await createStaleTab()
    await tab.save()

    expect(tab.isStale.value).toBe(true)
    expect(writes).toStrictEqual([])
  })
})
