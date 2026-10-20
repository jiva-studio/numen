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
    setSyncEnabled: async (kept: boolean) => {
      wrote.push(kept)
      return error
    },
  }
}

describe('whether a title and a filename are one name', () => {
  it('opens on what the settings hold, so the list stands on what is in force', async () => {
    const core = vault(false)
    const held = syncSetting(core, words, writer().write)
    await held.start()

    const groups = held.getSyncingGroups()
    expect(groups.map((group) => group.id)).toStrictEqual([SYNCING])
    const items = groups.flatMap((group) => group.items)
    expect(items.map((one) => one.id)).toStrictEqual([ON, OFF])
    expect(items.find((one) => one.isCurrent)?.id).toBe(OFF)
  })

  it('keeps the two one name where the vault cannot be asked', async () => {
    const held = syncSetting(
      {
        getSyncEnabled: async () => Promise.reject(new Error('no')),
        setSyncEnabled: async () => null,
      },
      words,
      writer().write,
    )
    await held.start()

    expect(held.kept.value).toBe(true)
  })

  it('says which of the two is the one in force, and nothing beside the other', async () => {
    const held = syncSetting(vault(true), words, writer().write)
    await held.start()

    const items = held.getSyncingGroups().flatMap((group) => group.items)
    expect(items.find((row) => row.id === ON)?.detail).toBe(words.current)
    expect(items.find((row) => row.id === OFF)?.detail).toBeUndefined()
  })

  it('writes the row chosen and stands on it', async () => {
    const core = vault(true)
    const held = syncSetting(core, words, writer().write)
    await held.start()

    await held.choose(OFF)

    expect(core.wrote).toStrictEqual([false])
    expect(held.kept.value).toBe(false)
  })

  it('writes nothing for the row already in force', async () => {
    const core = vault(true)
    const held = syncSetting(core, words, writer().write)
    await held.start()

    await held.choose(ON)

    expect(core.wrote).toStrictEqual([])
  })

  it('writes nothing for a row it does not offer', async () => {
    const core = vault(true)
    const held = syncSetting(core, words, writer().write)
    await held.start()

    await held.choose('interfaceScale:1.5')

    expect(core.wrote).toStrictEqual([])
    expect(held.kept.value).toBe(true)
  })

  it('goes back to what the settings hold where the setting could not be written', async () => {
    const core = vault(true, 'unreadable')
    const told = writer()
    const held = syncSetting(core, words, told.write)
    await held.start()

    await held.choose(OFF)

    expect(held.kept.value).toBe(true)
    expect(told.said.at(-1)).toContain(words.unturned)
  })

  it('writes the row chosen when the settings hold the other', async () => {
    const core = vault(false)
    const held = syncSetting(core, words, writer().write)
    await held.start()

    await held.choose(ON)

    expect(core.wrote).toStrictEqual([true])
    expect(held.kept.value).toBe(true)
  })

  it('clears the message line before it writes', async () => {
    const told = writer()
    const held = syncSetting(vault(true), words, told.write)
    await held.start()

    await held.choose(OFF)

    expect(told.told).toStrictEqual([{ text: '', kind: 'report' }])
  })

  it('says the failure as an error', async () => {
    const told = writer()
    const held = syncSetting(vault(true, 'unreadable'), words, told.write)
    await held.start()

    await held.choose(OFF)

    expect(told.told.at(-1)).toStrictEqual({
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

  it('stands on the last choice while the first write is refused', async () => {
    const core = slowVault()
    const held = syncSetting(core, words, writer().write)
    await held.start()

    const first = held.choose(OFF)
    const second = held.choose(ON)
    core.answers[0]?.('unreadable')
    await first
    expect(held.isEnabled.value).toBe(true)
    core.answers[1]?.(null)
    await second
    expect(held.isEnabled.value).toBe(true)
  })

  it('is not moved by a read of the settings made while a write is in flight', async () => {
    const core = slowVault()
    const held = syncSetting(core, words, writer().write)
    await held.start()

    const first = held.choose(OFF)
    await held.start()
    expect(held.isEnabled.value).toBe(false)
    core.answers[0]?.(null)
    await first
  })
})
