const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const
const BYTES_PER_UNIT = 1024

function getUnit(bytes: number): number {
  let size = Math.max(0, bytes)
  let unit = 0
  while (size >= BYTES_PER_UNIT && unit < BYTE_UNITS.length - 1) {
    size /= BYTES_PER_UNIT
    unit += 1
  }
  return unit
}

function formatFigure(bytes: number, unit: number, digits: number, locale?: string): string {
  return new Intl.NumberFormat(locale, {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(Math.max(0, bytes) / BYTES_PER_UNIT ** unit)
}

/** A size in bytes in the largest unit that keeps the figure at one or more, with one decimal. */
export function formatBytes(bytes: number, locale?: string): string {
  const unit = getUnit(bytes)
  return `${formatFigure(bytes, unit, unit === 0 ? 0 : 1, locale)} ${BYTE_UNITS[unit]}`
}

/** A part of a size, such as "12 of 41 MB": both figures in the unit of the whole, with up to one decimal. */
export function formatBytesOf(done: number, total: number, locale?: string): string {
  const unit = getUnit(total)
  const digits = unit === 0 ? 0 : 1
  const trim = (bytes: number): string =>
    formatFigure(bytes, unit, digits, locale).replace(/[.,]0$/, '')
  return `${trim(done)} of ${trim(total)} ${BYTE_UNITS[unit]}`
}
