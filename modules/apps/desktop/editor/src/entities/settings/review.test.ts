/**
 * The hour a day of review begins at, asked without a screen.
 *
 * The negatives are the ones worth having: an hour that could not be written
 * leaves the window standing where the settings do, and a vault that cannot be
 * asked leaves it standing at the hour an installation begins at.
 */
import { describe, expect, it, vi } from 'vitest'
import { reviewSetting, DEFAULT_STARTS, type ReviewDeps } from './review'

const words = { unturned: 'That setting could not be written:' }

/**
 * The day a vault counts from at one in the morning of the fifth, for each hour
 * a day of review may begin at. An hour the clock has not reached leaves the
 * fourth standing, and an hour it has passed opens the fifth.
 */
const DAYS: Record<string, string> = {
  '00:30': '2026-09-05',
  '01:00': '2026-09-05',
  '03:59': '2026-09-04',
  '04:00': '2026-09-04',
  '06:30': '2026-09-04',
}

/**
 * A vault holding that hour and counting from the day it names, and refusing
 * what it is told to refuse. An hour it takes is the hour it holds from then on.
 */
const vault = (starts: string, refuses: string | null = null, latest = '12:00') => {
  const written: string[] = []
  let hour = starts
  const core: ReviewDeps = {
    getReviewSettings: () => Promise.resolve({ starts: hour, latest, day: DAYS[hour] ?? '' }),
    setReviewSettings: (starts) => {
      written.push(starts)
      if (!refuses) hour = starts
      return Promise.resolve(refuses)
    },
  }
  const said = vi.fn()
  return { written, said, hours: reviewSetting(core, words, said) }
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
    const hours = reviewSetting(
      {
        getReviewSettings: () => Promise.reject(new Error('gone')),
        setReviewSettings: () => Promise.resolve(null),
      },
      words,
      said,
    )

    await hours.start()

    expect(hours.starts.value).toBe(DEFAULT_STARTS)
  })
})

describe('the review day now standing', () => {
  it('is nothing until the vault has been asked', () => {
    expect(vault('04:00').hours.day.value).toBe('')
  })

  it('is the day before while the hour in force has not come round', async () => {
    const { hours } = vault('04:00')
    await hours.start()
    expect(hours.day.value).toBe('2026-09-04')
  })

  it('is the day the calendar names where that hour has passed', async () => {
    const { hours } = vault('00:30')
    await hours.start()
    expect(hours.day.value).toBe('2026-09-05')
  })

  it('is asked again once an hour is written', async () => {
    const { hours } = vault('00:30')
    await hours.start()

    await hours.chooses('03:59')

    expect(hours.day.value).toBe('2026-09-04')
  })

  it('is left where it stands where the hour was refused', async () => {
    const { hours } = vault('00:30', 'the file could not be written')
    await hours.start()

    await hours.chooses('06:30')

    expect(hours.day.value).toBe('2026-09-05')
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
      'error',
    )
  })

  it('says what a person can read where the vault threw instead of answering', async () => {
    const said = vi.fn()
    const hours = reviewSetting(
      {
        getReviewSettings: () =>
          Promise.resolve({ starts: '04:00', latest: '12:00', day: '2026-09-04' }),
        setReviewSettings: () => Promise.reject(new Error('not an hour of the day')),
      },
      words,
      said,
    )

    await hours.chooses('19:00')

    expect(hours.starts.value).toBe(DEFAULT_STARTS)
    expect(said).toHaveBeenLastCalledWith(
      'That setting could not be written: numen did not answer, so nothing was done — it may have stopped, and the window keeps trying',
      'error',
    )
  })
})
