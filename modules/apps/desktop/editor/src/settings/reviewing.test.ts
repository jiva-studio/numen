/**
 * The hour a day of review begins at, asked without a screen.
 *
 * The negatives are the ones worth having: an hour that could not be written
 * leaves the window standing where the settings do, and a vault that cannot be
 * asked leaves it standing at the hour an installation begins at.
 */
import { describe, expect, it, vi } from 'vitest'
import { reviewing, DEFAULT_STARTS, type ReviewingDeps } from './reviewing'

const words = { unturned: 'That setting could not be written:' }

/** A vault holding that hour, and refusing what it is told to refuse. */
const vault = (held: string, refuses: string | null = null, latest = '12:00') => {
  const written: string[] = []
  const core: ReviewingDeps = {
    reviewing: () => Promise.resolve({ starts: held, latest }),
    choosesReviewing: (starts) => {
      written.push(starts)
      return Promise.resolve(refuses)
    },
  }
  const said = vi.fn()
  return { written, said, hours: reviewing(core, words, said) }
}

describe('the hour the window stands at', () => {
  it('is where an installation nobody has configured begins the day', () => {
    expect(vault('06:00').hours.starts.value).toBe(DEFAULT_STARTS)
  })

  it('is what the settings hold, once the vault has answered', async () => {
    const { hours } = vault('06:00')
    await hours.start()
    expect(hours.starts.value).toBe('06:00')
  })

  it('is left where it stands where the vault cannot be asked', async () => {
    const said = vi.fn()
    const hours = reviewing(
      {
        reviewing: () => Promise.reject(new Error('gone')),
        choosesReviewing: () => Promise.resolve(null),
      },
      words,
      said,
    )

    await hours.start()

    expect(hours.starts.value).toBe(DEFAULT_STARTS)
  })
})

describe('an hour chosen', () => {
  it('is written into the settings', async () => {
    const { hours, written } = vault('04:00')
    await hours.chooses('06:30')
    expect(written).toStrictEqual(['06:30'])
    expect(hours.starts.value).toBe('06:30')
  })

  it('is not written again where it is the hour in force', async () => {
    const { hours, written } = vault('04:00')
    await hours.chooses('04:00')
    expect(written).toStrictEqual([])
  })

  it('leaves the window where the settings are when it is refused', async () => {
    const { hours, said } = vault('04:00', 'the file could not be written')
    await hours.chooses('06:30')
    expect(hours.starts.value).toBe('04:00')
    expect(said).toHaveBeenLastCalledWith(
      'That setting could not be written: the file could not be written',
      'refusal',
    )
  })

  it('says what a person can read where the vault threw instead of answering', async () => {
    const said = vi.fn()
    const hours = reviewing(
      {
        reviewing: () => Promise.resolve({ starts: '04:00', latest: '12:00' }),
        choosesReviewing: () => Promise.reject(new Error('not an hour of the day')),
      },
      words,
      said,
    )

    await hours.chooses('19:00')

    expect(hours.starts.value).toBe(DEFAULT_STARTS)
    expect(said).toHaveBeenLastCalledWith(
      'That setting could not be written: numen did not answer, so nothing was done — it may have stopped, and the window keeps trying',
      'refusal',
    )
  })
})
