import { describe, expect, it } from 'vitest'

import { NESTING, OFF, ON, HEADINGS, useNestingSetting } from './nesting'
import { writer } from '@/testing/writer'
import { WORDS as words } from '@/shared/words'
import { DEFAULT_HEADINGS } from '@/entities/settings'

/** The vault, answering what the settings hold and keeping what was written. */
const vault = (isOn: boolean, headings = 6, error: string | null = null, most = 12) => {
  const wrote: boolean[] = []
  const counted: (number | undefined)[] = []
  return {
    wrote,
    counted,
    getNestingSettings: async () => ({ isNesting: isOn, headings, least: 1, most }),
    setNestingSettings: async (isNesting: boolean, count?: number) => {
      wrote.push(isNesting)
      counted.push(count)
      return error
    },
  }
}

describe('whether a node nests the headings of its note', () => {
  it('opens on what the settings hold, so the list relies on what is in force', async () => {
    const core = vault(false)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    const groups = target.getNestingGroups()
    expect(groups.map((group) => group.id)).toStrictEqual([NESTING])
    const items = groups.flatMap((group) => group.items)
    expect(items.map((one) => one.id)).toStrictEqual([ON, OFF])
    expect(items.find((one) => one.isCurrent)?.id).toBe(OFF)
  })

  it('nests the headings where the vault cannot be requested', async () => {
    const target = useNestingSetting(
      {
        getNestingSettings: async () => Promise.reject(new Error('no')),
        setNestingSettings: async () => null,
      },
      words,
      writer().write,
    )
    await target.start()

    expect(target.isNesting.value).toBe(true)
    expect(target.headings.value).toBe(6)
  })

  it('says which of the two is the one in force, and nothing alongside the other', async () => {
    const target = useNestingSetting(vault(true), words, writer().write)
    await target.start()

    const items = target.getNestingGroups().flatMap((group) => group.items)
    expect(items.find((row) => row.id === ON)?.detail).toBe(words.current)
    expect(items.find((row) => row.id === OFF)?.detail).toBeUndefined()
  })

  it('writes the row chosen and relies on it', async () => {
    const core = vault(true)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    await target.choose(OFF)

    expect(core.wrote).toStrictEqual([false])
    expect(target.isNesting.value).toBe(false)
  })

  it('writes nothing for the row already in force', async () => {
    const core = vault(true)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    await target.choose(ON)

    expect(core.wrote).toStrictEqual([])
  })

  it('writes nothing for a row it does not offer', async () => {
    const core = vault(true)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    await target.choose('interfaceScale:1.5')

    expect(core.wrote).toStrictEqual([])
    expect(target.isNesting.value).toBe(true)
  })

  it('goes back to what the settings hold where the setting could not be written', async () => {
    const core = vault(true, 6, 'unreadable')
    const notifiedCalls = writer()
    const target = useNestingSetting(core, words, notifiedCalls.write)
    await target.start()

    await target.choose(OFF)

    expect(target.isNesting.value).toBe(true)
    expect(notifiedCalls.messagesText.at(-1)).toContain(words.unturned)
  })
})

describe('how many headings stand under a node', () => {
  it('offers a ladder from one end of the setting to the other', async () => {
    const target = useNestingSetting(vault(true, 4), words, writer().write)
    await target.start()

    const groups = target.getHeadingsGroups()
    expect(groups.map((group) => group.id)).toStrictEqual([HEADINGS])
    const items = groups.flatMap((group) => group.items)
    expect(items.map((one) => one.id)).toStrictEqual(
      Array.from({ length: 12 }, (_, at) => `${at + 1}`),
    )
  })

  it('marks the count in force, and says nothing alongside the rest', async () => {
    const target = useNestingSetting(vault(true, 4), words, writer().write)
    await target.start()

    const items = target.getHeadingsGroups().flatMap((group) => group.items)
    expect(items.filter((one) => one.isCurrent).map((one) => one.id)).toStrictEqual(['4'])
    expect(items.find((one) => one.id === '4')?.detail).toBe(words.current)
    expect(items.find((one) => one.id === '5')?.detail).toBeUndefined()
  })

  it('writes the count chosen alongside the switch the window already knows', async () => {
    const core = vault(false, 6)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    await target.chooseCount('3')

    expect(core.wrote).toStrictEqual([false])
    expect(core.counted).toStrictEqual([3])
    expect(target.headings.value).toBe(3)
  })

  it('writes nothing for the count in force, or for one it does not offer', async () => {
    const core = vault(true, 6)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    await target.chooseCount('6')
    await target.chooseCount('13')
    await target.chooseCount('on')

    expect(core.counted).toStrictEqual([])
    expect(target.headings.value).toBe(6)
  })

  it('goes back to what the settings hold where the count could not be written', async () => {
    const core = vault(true, 6, 'unreadable')
    const notifiedCalls = writer()
    const target = useNestingSetting(core, words, notifiedCalls.write)
    await target.start()

    await target.chooseCount('3')

    expect(target.headings.value).toBe(6)
    expect(notifiedCalls.messagesText.at(-1)).toContain(words.unturned)
  })

  it('leaves the count out of a switch being turned', async () => {
    const core = vault(true, 4)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    await target.choose(OFF)

    expect(core.counted).toStrictEqual([undefined])
    expect(target.headings.value).toBe(4)
  })

  it('writes the row chosen when the settings hold the other', async () => {
    const core = vault(false)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    await target.choose(ON)

    expect(core.wrote).toStrictEqual([true])
    expect(target.isNesting.value).toBe(true)
  })

  it('clears the message line before it writes a switch', async () => {
    const notifiedCalls = writer()
    const target = useNestingSetting(vault(true), words, notifiedCalls.write)
    await target.start()

    await target.choose(OFF)

    expect(notifiedCalls.messages).toStrictEqual([{ text: '', kind: 'report' }])
  })

  it('says the failure of a switch as an error', async () => {
    const notifiedCalls = writer()
    const target = useNestingSetting(vault(true, 6, 'unreadable'), words, notifiedCalls.write)
    await target.start()

    await target.choose(OFF)

    expect(notifiedCalls.messages.at(-1)).toStrictEqual({
      text: `${words.unturned} unreadable`,
      kind: 'error',
    })
  })

  it('offers only the default count before the vault has returned how far the counts go', () => {
    const target = useNestingSetting(vault(true), words, writer().write)

    const items = target.getHeadingsGroups().flatMap((group) => group.items)
    expect(items.map((one) => one.id)).toStrictEqual([`${DEFAULT_HEADINGS}`])
  })

  it('titles each count with the number itself', async () => {
    const target = useNestingSetting(vault(true, 4), words, writer().write)
    await target.start()

    const items = target.getHeadingsGroups().flatMap((group) => group.items)
    expect(items.map((one) => one.title)).toStrictEqual(
      Array.from({ length: 12 }, (_, at) => `${at + 1}`),
    )
  })

  it('takes the counts at both ends of what the vault takes, and none beyond', async () => {
    const core = vault(true, 6)
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    await target.chooseCount('0')
    await target.chooseCount('1')
    await target.chooseCount('12')
    await target.chooseCount('13')

    expect(core.counted).toStrictEqual([1, 12])
    expect(target.headings.value).toBe(12)
  })

  it('clears the message line before it writes a count', async () => {
    const notifiedCalls = writer()
    const target = useNestingSetting(vault(true), words, notifiedCalls.write)
    await target.start()

    await target.chooseCount('3')

    expect(notifiedCalls.messages).toStrictEqual([{ text: '', kind: 'report' }])
  })

  it('says the failure of a count as an error', async () => {
    const notifiedCalls = writer()
    const target = useNestingSetting(vault(true, 6, 'unreadable'), words, notifiedCalls.write)
    await target.start()

    await target.chooseCount('3')

    expect(notifiedCalls.messages.at(-1)).toStrictEqual({
      text: `${words.unturned} unreadable`,
      kind: 'error',
    })
  })
})

describe('a switch turned twice in quick succession', () => {
  /** A vault whose writes are answered by hand, in the order a test chooses. */
  const slowVault = () => {
    const answers: ((failure: string | null) => void)[] = []
    return {
      answers,
      getNestingSettings: async () => ({ isNesting: true, headings: 6, least: 1, most: 12 }),
      setNestingSettings: () =>
        new Promise<string | null>((resolve) => {
          answers.push(resolve)
        }),
    }
  }

  it('relies on the last choice while the first write is still answered', async () => {
    const core = slowVault()
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    const first = target.choose(OFF)
    const second = target.choose(ON)
    core.answers[0]?.(null)
    await first
    expect(target.isNesting.value).toBe(true)
    core.answers[1]?.(null)
    await second
    expect(target.isNesting.value).toBe(true)
  })

  it('is not moved by a read of the settings made while a write is in flight', async () => {
    const core = slowVault()
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    const first = target.choose(OFF)
    await target.start()
    expect(target.isNesting.value).toBe(false)
    core.answers[0]?.(null)
    await first
  })

  it('keeps the last choice where an earlier write failed', async () => {
    const core = slowVault()
    const target = useNestingSetting(core, words, writer().write)
    await target.start()

    const first = target.choose(OFF)
    const second = target.choose(ON)
    core.answers[0]?.('unreadable')
    await first
    expect(target.isNesting.value).toBe(true)
    core.answers[1]?.(null)
    await second
  })
})
