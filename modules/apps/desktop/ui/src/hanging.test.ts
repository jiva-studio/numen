import { describe, expect, it } from 'vitest'

import { HANGING, OFF, ON, hanging } from './hanging'
import { WORDS as words } from './words'

/** The vault, answering what the settings hold and keeping what was written. */
const vault = (held: boolean, refuses: string | null = null) => {
  const wrote: boolean[] = []
  return {
    wrote,
    hanging: async () => held,
    choosesHanging: async (hangs: boolean) => {
      wrote.push(hangs)
      return refuses
    },
  }
}

/** What the window was told, in the order it was told.  */
const telling = () => {
  const said: string[] = []
  return { said, says: (text: string) => void said.push(text) }
}

describe('whether a node hangs the parts of its note', () => {
  it('opens on what the settings hold, so the list stands on what is in force', async () => {
    const core = vault(false)
    const held = hanging(core, words, telling().says)
    await held.start()

    const bands = held.offers()
    expect(bands.map((band) => band.id)).toStrictEqual([HANGING])
    const items = bands.flatMap((band) => band.items)
    expect(items.map((one) => one.id)).toStrictEqual([ON, OFF])
    expect(items.find((one) => one.inForce)?.id).toBe(OFF)
  })

  it('hangs the parts where the vault cannot be asked', async () => {
    const held = hanging(
      { hanging: async () => Promise.reject(new Error('no')), choosesHanging: async () => null },
      words,
      telling().says,
    )
    await held.start()

    expect(held.hangs.value).toBe(true)
  })

  it('says which of the two is the one in force, and nothing beside the other', async () => {
    const held = hanging(vault(true), words, telling().says)
    await held.start()

    const items = held.offers().flatMap((band) => band.items)
    expect(items.find((row) => row.id === ON)?.detail).toBe(words.current)
    expect(items.find((row) => row.id === OFF)?.detail).toBeUndefined()
  })

  it('writes the row chosen and stands on it', async () => {
    const core = vault(true)
    const held = hanging(core, words, telling().says)
    await held.start()

    await held.chooses(OFF)

    expect(core.wrote).toStrictEqual([false])
    expect(held.hangs.value).toBe(false)
  })

  it('writes nothing for the row already in force', async () => {
    const core = vault(true)
    const held = hanging(core, words, telling().says)
    await held.start()

    await held.chooses(ON)

    expect(core.wrote).toStrictEqual([])
  })

  it('writes nothing for a row it does not offer', async () => {
    const core = vault(true)
    const held = hanging(core, words, telling().says)
    await held.start()

    await held.chooses('interfaceScale:1.5')

    expect(core.wrote).toStrictEqual([])
    expect(held.hangs.value).toBe(true)
  })

  it('goes back to what the settings hold where the setting could not be written', async () => {
    const core = vault(true, 'unreadable')
    const told = telling()
    const held = hanging(core, words, told.says)
    await held.start()

    await held.chooses(OFF)

    expect(held.hangs.value).toBe(true)
    expect(told.said.at(-1)).toContain(words.unturned)
  })
})
