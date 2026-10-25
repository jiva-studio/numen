import { describe, expect, it } from 'vitest'

import { OFF, ON, SYNCING, syncSetting } from './sync'
import { writer } from '@/testing/writer'
import { WORDS as words } from '@/shared/words'

/** The vault, answering what the settings hold and keeping what was written. */
const vault = (isOn: boolean, error: string | null = null) => {
  const wrote: boolean[] = []
  return {
    wrote,
    getSyncEnabled: async () => isOn,
    setSyncEnabled: async (retained: boolean) => {
      wrote.push(retained)
      return error
    },
  }
}

describe('whether a title and a filename are one name', () => {
  it('opens on what the settings hold, so the list relies on what is in force', async () => {
    const core = vault(false)
    const target = syncSetting(core, words, writer().write)
    await target.start()

    const groups = target.getSyncingGroups()
    expect(groups.map((group) => group.id)).toStrictEqual([SYNCING])
    const items = groups.flatMap((group) => group.items)
    expect(items.map((one) => one.id)).toStrictEqual([ON, OFF])
    expect(items.find((one) => one.isCurrent)?.id).toBe(OFF)
  })

  it('keeps the two one name where the vault cannot be requested', async () => {
    const target = syncSetting(
      {
        getSyncEnabled: async () => Promise.reject(new Error('no')),
        setSyncEnabled: async () => null,
      },
      words,
      writer().write,
    )
    await target.start()

    expect(target.isEnabled.value).toBe(true)
  })

  it('says which of the two is the one in force, and nothing alongside the other', async () => {
    const target = syncSetting(vault(true), words, writer().write)
    await target.start()

    const items = target.getSyncingGroups().flatMap((group) => group.items)
    expect(items.find((row) => row.id === ON)?.detail).toBe(words.current)
    expect(items.find((row) => row.id === OFF)?.detail).toBeUndefined()
  })

  it('writes the row chosen and relies on it', async () => {
    const core = vault(true)
    const target = syncSetting(core, words, writer().write)
    await target.start()

    await target.choose(OFF)

    expect(core.wrote).toStrictEqual([false])
    expect(target.isEnabled.value).toBe(false)
  })

  it('writes nothing for the row already in force', async () => {
    const core = vault(true)
    const target = syncSetting(core, words, writer().write)
    await target.start()

    await target.choose(ON)

    expect(core.wrote).toStrictEqual([])
  })

  it('writes nothing for a row it does not offer', async () => {
    const core = vault(true)
    const target = syncSetting(core, words, writer().write)
    await target.start()

    await target.choose('interfaceScale:1.5')

    expect(core.wrote).toStrictEqual([])
    expect(target.isEnabled.value).toBe(true)
  })

  it('goes back to what the settings hold where the setting could not be written', async () => {
    const core = vault(true, 'unreadable')
    const mockWriter = writer()
    const target = syncSetting(core, words, mockWriter.write)
    await target.start()

    await target.choose(OFF)

    expect(target.isEnabled.value).toBe(true)
    expect(mockWriter.messagesText.at(-1)).toContain(words.unturned)
  })

  it('writes the row chosen when the settings hold the other', async () => {
    const core = vault(false)
    const target = syncSetting(core, words, writer().write)
    await target.start()

    await target.choose(ON)

    expect(core.wrote).toStrictEqual([true])
    expect(target.isEnabled.value).toBe(true)
  })

  it('clears the message line before it writes', async () => {
    const mockWriter = writer()
    const target = syncSetting(vault(true), words, mockWriter.write)
    await target.start()

    await target.choose(OFF)

    expect(mockWriter.messages).toStrictEqual([{ text: '', kind: 'report' }])
  })

  it('says the failure as an error', async () => {
    const mockWriter = writer()
    const target = syncSetting(vault(true, 'unreadable'), words, mockWriter.write)
    await target.start()

    await target.choose(OFF)

    expect(mockWriter.messages.at(-1)).toStrictEqual({
      text: `${words.unturned} unreadable`,
      kind: 'error',
    })
  })
})

describe('a switch turned twice in quick succession', () => {
  const slowVault = () => {
    const answers: ((failure: string | null) => void)[] = []
    return {
      answers,
      getSyncEnabled: async () => true,
      setSyncEnabled: () =>
        new Promise<string | null>((resolve) => {
          answers.push(resolve)
        }),
    }
  }

  it('relies on the last choice while the first write fails', async () => {
    const core = slowVault()
    const target = syncSetting(core, words, writer().write)
    await target.start()

    const first = target.choose(OFF)
    const second = target.choose(ON)
    core.answers[0]?.('unreadable')
    await first
    expect(target.isEnabled.value).toBe(true)
    core.answers[1]?.(null)
    await second
    expect(target.isEnabled.value).toBe(true)
  })

  it('is not moved by a read of the settings made while a write is in flight', async () => {
    const core = slowVault()
    const target = syncSetting(core, words, writer().write)
    await target.start()

    const first = target.choose(OFF)
    await target.start()
    expect(target.isEnabled.value).toBe(false)
    core.answers[0]?.(null)
    await first
  })
})
