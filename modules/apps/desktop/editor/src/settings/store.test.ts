/**
 * The settings read out of the file whole, asked without a screen.
 *
 * The negatives are the ones worth having: a vault that cannot be asked leaves
 * the window holding nothing, and a write that was refused is said and the
 * window goes back to what the settings hold.
 */
import { describe, expect, it, vi } from 'vitest'
import type { Model } from '../shared/core'
import { settingsStore, settingAt, type SettingsStoreDeps } from './store'

const words = {
  unturned: 'That setting could not be written:',
  unreadSettings: 'The settings could not be read.',
}

const MODELS: readonly Model[] = [
  {
    namedAt: ['agent', 'claude', 'model'],
    name: 'opus',
    title: 'opus',
    shelf: '',
    byDefault: false,
    writes: [{ at: ['agent', 'claude', 'model'], value: '"opus"' }],
    presence: 'nothing to fetch',
  },
]

/** A vault holding those settings, and refusing what it is told to refuse. */
const holding = (written: string, refuses: string | null = null) => {
  const asked: unknown[] = []
  const core: SettingsStoreDeps = {
    settings: () => Promise.resolve({ written, path: '/numen.json', models: MODELS }),
    choosesSetting: (said) => {
      asked.push(said)
      return refuses ? Promise.reject(new Error(refuses)) : Promise.resolve()
    },
  }
  const said = vi.fn()
  return { asked, said, kept: settingsStore(core, words, said) }
}

describe('what stands at a setting', () => {
  it('is what the file holds, once the vault has answered', async () => {
    const { kept } = holding('{"agent": {"claude": {"model": "opus"}}}')
    await kept.start()
    expect(kept.at(['agent', 'claude', 'model'])).toBe('opus')
    expect(kept.path.value).toBe('/numen.json')
  })

  it('is nothing where the file names it nowhere', async () => {
    const { kept } = holding('{}')
    await kept.start()
    expect(kept.at(['agent', 'claude', 'model'])).toBeUndefined()
  })

  it('is nothing where the vault cannot be asked', async () => {
    const said = vi.fn()
    const kept = settingsStore(
      { settings: () => Promise.reject(new Error('gone')), choosesSetting: () => Promise.resolve() },
      words,
      said,
    )

    await kept.start()

    expect(kept.at(['agent'])).toBeUndefined()
    expect(kept.path.value).toBe('')
  })

  it('is nothing where the vault answers with what is not JSON, and is said', async () => {
    const { kept, said } = holding('not JSON at all')
    await kept.start()
    expect(kept.at(['agent'])).toBeUndefined()
    expect(said).toHaveBeenLastCalledWith('The settings could not be read.', 'refusal')
  })
})

describe('the models a setting offers', () => {
  it('are the ones read from that setting, and no others', async () => {
    const { kept } = holding('{}')
    await kept.start()
    expect(kept.offers(['agent', 'claude', 'model']).map((one) => one.name)).toStrictEqual(['opus'])
    expect(kept.offers(['agent', 'use'])).toStrictEqual([])
  })
})

describe('a setting written', () => {
  it('is asked of the vault as it was given', async () => {
    const { kept, asked } = holding('{}')
    await kept.puts(['agent', 'use'], 'claude')
    expect(asked).toStrictEqual([[{ at: ['agent', 'use'], value: '"claude"' }]])
  })

  it('asks nothing of the vault where there is nothing to write', async () => {
    const { kept, asked } = holding('{}')
    await kept.chooses([])
    expect(asked).toStrictEqual([])
  })

  it('is said where it was refused', async () => {
    const { kept, said } = holding('{}', 'the file could not be written')
    await kept.puts(['agent', 'use'], 'claude')
    expect(said).toHaveBeenLastCalledWith(
      'That setting could not be written: numen did not answer, so nothing was done — it may have stopped, and the window keeps trying',
      'refusal',
    )
  })
})

describe('what stands at a path through a tree', () => {
  it('is the value the path leads to', () => {
    expect(settingAt({ a: { b: [1, 2] } }, ['a', 'b'])).toStrictEqual([1, 2])
  })

  it('is the tree itself for a path of no steps', () => {
    expect(settingAt({ a: 1 }, [])).toStrictEqual({ a: 1 })
  })

  it('is nothing where the path runs off the tree', () => {
    expect(settingAt({ a: 1 }, ['a', 'b'])).toBeUndefined()
    expect(settingAt(null, ['a'])).toBeUndefined()
  })
})
