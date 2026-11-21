export interface DateShortcut {
  readonly label: string
  readonly date: string
}

const WEEKDAYS = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday']
const DAY_MS = 86_400_000

function parse(date: string): Date {
  return new Date(`${date}T00:00:00Z`)
}

function format(day: Date): string {
  return day.toISOString().slice(0, 10)
}

function addDays(date: string, days: number): string {
  return format(new Date(parse(date).getTime() + days * DAY_MS))
}

function addMonths(date: string, months: number): string {
  const day = parse(date)
  const target = new Date(Date.UTC(day.getUTCFullYear(), day.getUTCMonth() + months, 1))
  const last = new Date(Date.UTC(target.getUTCFullYear(), target.getUTCMonth() + 1, 0)).getUTCDate()
  target.setUTCDate(Math.min(day.getUTCDate(), last))
  return format(target)
}

/** The days a person types as a word, counted from today (`YYYY-MM-DD`). */
export function listDateShortcuts(today: string): DateShortcut[] {
  const weekday = parse(today).getUTCDay()
  const weekdays = Array.from({ length: 7 }, (_, at) => {
    const ahead = at + 1
    return { label: WEEKDAYS[(weekday + ahead) % 7] ?? '', date: addDays(today, ahead) }
  })
  return [
    { label: 'today', date: today },
    { label: 'tomorrow', date: addDays(today, 1) },
    ...weekdays.slice(1),
    { label: '+1w', date: addDays(today, 7) },
    { label: '+2w', date: addDays(today, 14) },
    { label: '+1m', date: addMonths(today, 1) },
  ]
}
