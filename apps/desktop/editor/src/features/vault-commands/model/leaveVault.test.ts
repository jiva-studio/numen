import { describe, expect, it } from 'vitest'
import { openAfterSavingTabs } from './leaveVault'

const record = () => {
  const calls: string[] = []
  const hold = {
    pause: async () => {
      await Promise.resolve()
      calls.push('pause')
    },
    resume: () => calls.push('resume'),
  }
  return { calls, hold }
}

describe('opening a vault', () => {
  it('writes the tabs to the vault being left before the core shows the other one', async () => {
    const { calls, hold } = record()
    const open = openAfterSavingTabs(async (id) => {
      calls.push(`open ${id}`)
      return { ok: true, value: null }
    }, hold)

    expect(await open('second')).toStrictEqual({ ok: true, value: null })

    expect(calls).toStrictEqual(['pause', 'open second'])
  })

  it('writes the tabs again when the core refuses the vault', async () => {
    const { calls, hold } = record()
    const open = openAfterSavingTabs(async () => ({ ok: false, error: 'unreadable' }), hold)

    expect(await open('second')).toStrictEqual({ ok: false, error: 'unreadable' })

    expect(calls).toStrictEqual(['pause', 'resume'])
  })

  it('writes the tabs again when the call fails', async () => {
    const { calls, hold } = record()
    const open = openAfterSavingTabs(async () => {
      throw new Error('window closing')
    }, hold)

    await expect(open('second')).rejects.toThrow('window closing')

    expect(calls).toStrictEqual(['pause', 'resume'])
  })
})
