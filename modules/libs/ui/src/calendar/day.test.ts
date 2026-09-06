/**
 * A day of the calendar, counted on the calendar.
 *
 * Where a zone or a clock change would tell, the machine's clock is moved under
 * the test: an hour put into a clock or taken out of one is not a day, and
 * neither is a zone that is not Greenwich.
 */
import { describe, expect, it } from 'vitest'
import { dayAfter, dayNamed, dayOf, daysBetween, isDay } from './day'

/**
 * The zone the machine stands in, as vitest moves it. A test that moves it puts
 * it back, since every other test here reads it.
 */
const inZone = (zone: string, run: () => void) => {
  const was = process.env['TZ']
  process.env['TZ'] = zone
  try {
    run()
  } finally {
    if (was === undefined) delete process.env['TZ']
    else process.env['TZ'] = was
  }
}

describe('a day as it is written down', () => {
  it('is the year, the month and the day, each at its width', () => {
    expect(dayNamed(new Date(2026, 0, 5))).toBe('2026-01-05')
    expect(dayNamed(new Date(2026, 11, 31))).toBe('2026-12-31')
  })

  // A machine east or west of Greenwich reads a different date from the one in
  // London for part of every day, and the calendar a person is looking at is
  // their own.
  it('is the day on the person’s own calendar, not the day at Greenwich', () => {
    inZone('America/New_York', () => {
      expect(dayNamed(new Date(2026, 8, 5, 23, 30))).toBe('2026-09-05')
    })
    inZone('Pacific/Auckland', () => {
      expect(dayNamed(new Date(2026, 8, 5, 0, 30))).toBe('2026-09-05')
    })
  })
})

describe('the instant a written day is read at', () => {
  it('is noon, which no clock change moves off the day', () => {
    const at = dayOf('2026-03-29')

    expect(at.getFullYear()).toBe(2026)
    expect(at.getMonth()).toBe(2)
    expect(at.getDate()).toBe(29)
    expect(at.getHours()).toBe(12)
  })
})

describe('whether a value is a day', () => {
  it('is true of a day and false of a field that holds no date', () => {
    expect(isDay('2026-09-05')).toBe(true)
    expect(isDay('')).toBe(false)
    expect(isDay('soon')).toBe(false)
  })
})

describe('the days between two written days', () => {
  it('counts them on the calendar, both ways', () => {
    expect(daysBetween('2026-09-05', '2026-09-12')).toBe(7)
    expect(daysBetween('2026-09-12', '2026-09-05')).toBe(-7)
    expect(daysBetween('2026-09-05', '2026-09-05')).toBe(0)
  })

  it('counts a leap day as a day', () => {
    expect(daysBetween('2028-02-28', '2028-03-01')).toBe(2)
    expect(daysBetween('2027-02-28', '2027-03-01')).toBe(1)
  })

  // The clock goes forward on 29 March 2026 in Berlin and back on 25 October,
  // and neither is a day gained or lost.
  it('is the same across a clock going forward and a clock going back', () => {
    inZone('Europe/Berlin', () => {
      expect(daysBetween('2026-03-28', '2026-03-30')).toBe(2)
      expect(daysBetween('2026-10-24', '2026-10-26')).toBe(2)
      expect(daysBetween('2026-01-01', '2026-12-31')).toBe(364)
    })
  })

  it('is none where either end is not a day', () => {
    expect(daysBetween('', '2026-09-05')).toBe(0)
    expect(daysBetween('2026-09-05', '')).toBe(0)
  })
})

describe('the day so many days after another', () => {
  it('walks the calendar, forward and back', () => {
    expect(dayAfter('2026-09-05', 7)).toBe('2026-09-12')
    expect(dayAfter('2026-09-05', -7)).toBe('2026-08-29')
    expect(dayAfter('2026-09-05', 0)).toBe('2026-09-05')
  })

  it('walks over a month, a year and a leap day', () => {
    expect(dayAfter('2026-12-31', 1)).toBe('2027-01-01')
    expect(dayAfter('2028-02-28', 1)).toBe('2028-02-29')
  })

  it('walks over a clock change without gaining or losing a day', () => {
    inZone('Europe/Berlin', () => {
      expect(dayAfter('2026-03-28', 1)).toBe('2026-03-29')
      expect(dayAfter('2026-03-29', 1)).toBe('2026-03-30')
      expect(dayAfter('2026-10-25', 1)).toBe('2026-10-26')
    })
  })

  it('is the value itself where it is not a day', () => {
    expect(dayAfter('', 3)).toBe('')
  })
})
