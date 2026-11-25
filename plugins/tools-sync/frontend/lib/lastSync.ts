import { WORDS as words } from '../words'

const MS_PER_MINUTE = 60_000
const MINUTES_PER_HOUR = 60
const HOURS_PER_DAY = 24
const DAYS_PER_WEEK = 7

const dateFormat = new Intl.DateTimeFormat('en', { day: 'numeric', month: 'short', year: 'numeric' })

/** The time since a sync in plain words: minutes, hours, days, and a date from a week on. */
export function formatLastSync(syncedAt: Date, now: Date): string {
  const minutes = Math.floor((now.getTime() - syncedAt.getTime()) / MS_PER_MINUTE)
  if (minutes < 1) return words.justNow
  if (minutes < MINUTES_PER_HOUR) return words.minutesAgo(minutes)
  const hours = Math.floor(minutes / MINUTES_PER_HOUR)
  if (hours < HOURS_PER_DAY) return words.hoursAgo(hours)
  const days = Math.floor(hours / HOURS_PER_DAY)
  if (days === 1) return words.yesterday
  if (days < DAYS_PER_WEEK) return words.daysAgo(days)
  return dateFormat.format(syncedAt)
}
