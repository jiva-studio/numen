import { describe, expect, it } from 'vitest'

import { OFF, ON, SYNCING, syncSetting } from './sync'
import { writer } from '@/testing/writer'
import { WORDS as words } from '@/shared/words'

/** The vault, answering what the settings hold and keeping what was written. */
const vault = (held: boolean, refuses: string | null = null) => {
  const wrote: boolean[] = []
  return {
    wrote,
    getSyncEnabled: async () => held,
    setSyncEnabled: async (kept: boolean) => {
      wrote.push(kept)
      return refuses
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
    expect(items.find((one) => one.inForce)?.id).toBe(OFF)
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

    await held.chooses(OFF)

    expect(core.wrote).toStrictEqual([false])
    expect(held.kept.value).toBe(false)
  })

  it('writes nothing for the row already in force', async () => {
    const core = vault(true)
    const held = syncSetting(core, words, writer().write)
    await held.start()

    await held.chooses(ON)

    expect(core.wrote).toStrictEqual([])
  })

  it('writes nothing for a row it does not offer', async () => {
    const core = vault(true)
    const held = syncSetting(core, words, writer().write)
    await held.start()

    await held.chooses('interfaceScale:1.5')

    expect(core.wrote).toStrictEqual([])
    expect(held.kept.value).toBe(true)
  })

  it('goes back to what the settings hold where the setting could not be written', async () => {
    const core = vault(true, 'unreadable')
    const told = writer()
    const held = syncSetting(core, words, told.write)
    await held.start()

    await held.chooses(OFF)

    expect(held.kept.value).toBe(true)
    expect(told.said.at(-1)).toContain(words.unturned)
  })
})
