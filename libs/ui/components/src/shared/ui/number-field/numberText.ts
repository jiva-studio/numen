/**
 * What a line of typing evaluates to as a number, where the bounds leave it, and
 * where a key leaves it.
 *
 * Pure: the same text gives the same answer wherever it is read, so what the
 * field hands on can be named without drawing it.
 */
import type { Bounds } from '@/shared/lib/math/bounds'
import { fitToBounds } from './fitToBounds'
import { stepBy } from './stepBy'

/** Text a number could still be typed out of: a lone sign, a trailing point. */
const ON_ITS_WAY = /^[+-]?(\d+([.,]\d*)?|[.,]\d*)?$/

/** A comma between a whole number and a group of three, which is no point. */
const GROUPED = /^[+-]?[1-9]\d{0,2},\d{3}$/

/** Whether more typing could still make a number of the text. */
export const isPartialNumber = (text: string): boolean => {
  const cleanText = text.trim()
  return ON_ITS_WAY.test(cleanText) && !GROUPED.test(cleanText)
}

/**
 * The number the text represents; null when it does not represent one. The
 * accepted shape is what a user types: no exponent, no hexadecimal, no word for infinity.
 */
export const parseNumber = (text: string): number | null => {
  const cleanText = text.trim()
  if (cleanText === '' || !isPartialNumber(cleanText)) return null
  const value = Number(cleanText.replace(',', '.'))
  return Number.isFinite(value) ? value : null
}

/** Whether the text is a number the bounds allow. An empty field is neither. */
export const isAllowed = (text: string, bounds: Bounds): boolean => {
  const value = parseNumber(text)
  // Stryker disable next-line ConditionalExpression: value !== null narrows number | null for TypeScript
  return value !== null && value === fitToBounds(value, bounds)
}

/** How many steps a page key covers at once. */
const PACES = 10

/** Where a key leaves the number, and nothing for a key the field does not answer. */
export const stepForKey = (key: string, value: number | null, bounds: Bounds): number | null => {
  if (key === 'Home') return fitToBounds(bounds.min, bounds)
  if (key === 'End') return fitToBounds(bounds.max, bounds)
  if (key === 'ArrowUp') return stepBy(value, 1, bounds)
  if (key === 'ArrowDown') return stepBy(value, -1, bounds)
  if (key === 'PageUp') return stepBy(value, PACES, bounds)
  if (key === 'PageDown') return stepBy(value, -PACES, bounds)
  return null
}

/** How a number is written into the field. */
export const formatNumber = (value: number | null): string => (value === null ? '' : String(value))

/** Whether the text represents the current value. An empty field represents null. */
export const isTextForValue = (text: string, value: number | null): boolean =>
  value === null ? text.trim() === '' : parseNumber(text) === value
