import { describe, expect, it } from 'vitest'
import { listDateShortcuts } from './dateShortcuts'

const at = (today: string) =>
  Object.fromEntries(listDateShortcuts(today).map((one) => [one.label, one.date]))

describe('the date shortcuts', () => {
  it('count today, tomorrow, the next weekdays and weeks from the day they are given', () => {
    expect(at('2026-11-21')).toEqual({
      today: '2026-11-21',
      tomorrow: '2026-11-22',
      monday: '2026-11-23',
      tuesday: '2026-11-24',
      wednesday: '2026-11-25',
      thursday: '2026-11-26',
      friday: '2026-11-27',
      saturday: '2026-11-28',
      '+1w': '2026-11-28',
      '+2w': '2026-12-05',
      '+1m': '2026-12-21',
    })
  })

  it('name the weekday after tomorrow in the order they come and a week ahead for the same weekday', () => {
    const labels = listDateShortcuts('2026-11-21').map((one) => one.label)
    expect(labels.slice(0, 4)).toEqual(['today', 'tomorrow', 'monday', 'tuesday'])
    expect(labels).toContain('saturday')
  })

  it('cross a year and keep a month end inside the next month', () => {
    expect(at('2026-12-31')['tomorrow']).toBe('2027-01-01')
    expect(at('2026-01-31')['+1m']).toBe('2026-02-28')
  })
})
