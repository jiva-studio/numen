import { describe, expect, it } from 'vitest'

import { counting } from './counting'
import type { Counted, Counts } from './counting'

const vault = (id: string, said: Partial<Counted['vaults'][number]> = {}) => ({
  vaultId: id,
  name: id,
  path: `/vaults/${id}`,
  faces: 3,
  due: 1,
  new: 2,
  decks: [{ deck: 'decks/Words.md', faces: 3, due: 1, new: 2 }],
  unread: '',
  ...said,
})

describe('counting what every vault owes', () => {
  it('holds what the front door answered', async () => {
    const cards: Counts = { async owing() { return { vaults: [vault('01A'), vault('01B')] } } }
    const one = counting({ cards, failed: () => {} })

    await one.count()

    expect(one.vaults.value).toHaveLength(2)
    expect(one.vaults.value[0]).toMatchObject({ vaultId: '01A', due: 1, new: 2 })
    expect(one.counting.value).toBe(false)
  })

  // The window opening, a sitting ending and a vault moving underneath it all
  // ask, and they arrive together. One count answers all three.
  it('runs one count however many ask for it at once', async () => {
    let asked = 0
    let settle = (_: Counted) => {}
    const cards: Counts = {
      owing() {
        asked += 1
        return new Promise<Counted>((then) => {
          settle = then
        })
      },
    }
    const one = counting({ cards, failed: () => {} })

    const three = [one.count(), one.count(), one.count()]
    expect(asked).toBe(1)

    settle({ vaults: [vault('01A')] })
    await Promise.all(three)
    expect(one.vaults.value).toHaveLength(1)

    // And the next ask is a count of its own, because the vaults have moved on.
    settle = () => {}
    void one.count()
    expect(asked).toBe(2)
  })

  it('says what went wrong and stops counting', async () => {
    const trouble: unknown[] = []
    const cards: Counts = { owing: () => Promise.reject(new Error('no registry')) }
    const one = counting({ cards, failed: (why) => trouble.push(why) })

    await one.count()

    expect(trouble).toHaveLength(1)
    expect(one.counting.value).toBe(false)
    expect(one.vaults.value).toHaveLength(0)
  })

  it('carries what a vault that could not be counted says', async () => {
    const cards: Counts = {
      async owing() {
        return { vaults: [vault('01A', { unread: 'this vault has not been read yet', faces: 0, due: 0, new: 0 })] }
      },
    }
    const one = counting({ cards, failed: () => {} })

    await one.count()

    expect(one.vaults.value[0]?.unread).toBe('this vault has not been read yet')
  })
})
