import { describe, expect, it } from 'vitest'

import { HANGING, OFF, ON, PARTS, useHangingSetting } from './hanging'
import { writer } from '@/testing/writer'
import { WORDS as words } from '@/shared/words'

/** The vault, answering what the settings hold and keeping what was written. */
const vault = (held: boolean, parts = 6, refuses: string | null = null, most = 12) => {
  const wrote: boolean[] = []
  const counted: (number | undefined)[] = []
  return {
    wrote,
    counted,
    getHangingSettings: async () => ({ hangs: held, parts, least: 1, most }),
    setHangingSettings: async (hangs: boolean, count?: number) => {
      wrote.push(hangs)
      counted.push(count)
      return refuses
    },
  }
}

describe('whether a node hangs the parts of its note', () => {
  it('opens on what the settings hold, so the list stands on what is in force', async () => {
    const core = vault(false)
    const held = useHangingSetting(core, words, writer().says)
    await held.start()

    const groups = held.offers()
    expect(groups.map((group) => group.id)).toStrictEqual([HANGING])
    const items = groups.flatMap((group) => group.items)
    expect(items.map((one) => one.id)).toStrictEqual([ON, OFF])
    expect(items.find((one) => one.inForce)?.id).toBe(OFF)
  })

  it('hangs the parts where the vault cannot be asked', async () => {
    const held = useHangingSetting(
      {
        getHangingSettings: async () => Promise.reject(new Error('no')),
        setHangingSettings: async () => null,
      },
      words,
      writer().says,
    )
    await held.start()

    expect(held.hangs.value).toBe(true)
    expect(held.parts.value).toBe(6)
  })

  it('says which of the two is the one in force, and nothing beside the other', async () => {
    const held = useHangingSetting(vault(true), words, writer().says)
    await held.start()

    const items = held.offers().flatMap((group) => group.items)
    expect(items.find((row) => row.id === ON)?.detail).toBe(words.current)
    expect(items.find((row) => row.id === OFF)?.detail).toBeUndefined()
  })

  it('writes the row chosen and stands on it', async () => {
    const core = vault(true)
    const held = useHangingSetting(core, words, writer().says)
    await held.start()

    await held.chooses(OFF)

    expect(core.wrote).toStrictEqual([false])
    expect(held.hangs.value).toBe(false)
  })

  it('writes nothing for the row already in force', async () => {
    const core = vault(true)
    const held = useHangingSetting(core, words, writer().says)
    await held.start()

    await held.chooses(ON)

    expect(core.wrote).toStrictEqual([])
  })

  it('writes nothing for a row it does not offer', async () => {
    const core = vault(true)
    const held = useHangingSetting(core, words, writer().says)
    await held.start()

    await held.chooses('interfaceScale:1.5')

    expect(core.wrote).toStrictEqual([])
    expect(held.hangs.value).toBe(true)
  })

  it('goes back to what the settings hold where the setting could not be written', async () => {
    const core = vault(true, 6, 'unreadable')
    const told = writer()
    const held = useHangingSetting(core, words, told.says)
    await held.start()

    await held.chooses(OFF)

    expect(held.hangs.value).toBe(true)
    expect(told.said.at(-1)).toContain(words.unturned)
  })
})

describe('how many parts stand under a node', () => {
  it('offers a ladder from one end of the setting to the other', async () => {
    const held = useHangingSetting(vault(true, 4), words, writer().says)
    await held.start()

    const groups = held.counts()
    expect(groups.map((group) => group.id)).toStrictEqual([PARTS])
    const items = groups.flatMap((group) => group.items)
    expect(items.map((one) => one.id)).toStrictEqual(
      Array.from({ length: 12 }, (_, at) => `${at + 1}`),
    )
  })

  it('marks the count in force, and says nothing beside the rest', async () => {
    const held = useHangingSetting(vault(true, 4), words, writer().says)
    await held.start()

    const items = held.counts().flatMap((group) => group.items)
    expect(items.filter((one) => one.inForce).map((one) => one.id)).toStrictEqual(['4'])
    expect(items.find((one) => one.id === '4')?.detail).toBe(words.current)
    expect(items.find((one) => one.id === '5')?.detail).toBeUndefined()
  })

  it('writes the count chosen beside the switch the window already knows', async () => {
    const core = vault(false, 6)
    const held = useHangingSetting(core, words, writer().says)
    await held.start()

    await held.choosesCount('3')

    expect(core.wrote).toStrictEqual([false])
    expect(core.counted).toStrictEqual([3])
    expect(held.parts.value).toBe(3)
  })

  it('writes nothing for the count in force, or for one it does not offer', async () => {
    const core = vault(true, 6)
    const held = useHangingSetting(core, words, writer().says)
    await held.start()

    await held.choosesCount('6')
    await held.choosesCount('13')
    await held.choosesCount('on')

    expect(core.counted).toStrictEqual([])
    expect(held.parts.value).toBe(6)
  })

  it('goes back to what the settings hold where the count could not be written', async () => {
    const core = vault(true, 6, 'unreadable')
    const told = writer()
    const held = useHangingSetting(core, words, told.says)
    await held.start()

    await held.choosesCount('3')

    expect(held.parts.value).toBe(6)
    expect(told.said.at(-1)).toContain(words.unturned)
  })

  it('leaves the count out of a switch being turned', async () => {
    const core = vault(true, 4)
    const held = useHangingSetting(core, words, writer().says)
    await held.start()

    await held.chooses(OFF)

    expect(core.counted).toStrictEqual([undefined])
    expect(held.parts.value).toBe(4)
  })
})
