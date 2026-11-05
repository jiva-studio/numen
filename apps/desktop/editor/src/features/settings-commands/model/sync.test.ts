import { describe, expect, it } from 'vitest'

import { OFF, ON, syncSetting } from './sync'
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
  it('reads what the settings hold, so the window relies on what is in force', async () => {
    const target = syncSetting(vault(false), words, writer().write)
    await target.start()

    expect(target.isEnabled.value).toBe(false)
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

    expect(mockWriter.messages).toStrictEqual([{ text: '', kind: 'info' }])
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

  it('is not moved by a read that began before a write and answers after it', async () => {
    let answerRead: (isOn: boolean) => void = () => undefined
    const core = {
      getSyncEnabled: () => new Promise<boolean>((resolve) => (answerRead = resolve)),
      setSyncEnabled: async () => null,
    }
    const target = syncSetting(core, words, writer().write)
    const read = target.start()

    await target.choose(OFF)
    answerRead(true)
    await read

    expect(target.isEnabled.value).toBe(false)
  })
})
