/**
 * Number formatting for display. A count of text reaches six figures in an
 * ordinary vault, and the grouping is identical wherever it is rendered.
 */

/**
 * A share as a percentage, not a fraction.
 *
 * Rounded, so a half is read as fifty and a whole as a hundred. A count that
 * has still to finish reads by `percentWord`, which never rounds up to a
 * hundred.
 */
export const percent = (share: number): string => `${Math.round(share * 100)}%`

/**
 * The thing a count counts, in the singular where there is one of it. `other`
 * is the whole plural where an `s` on the end does not give it, and the two
 * arguments are the two forms English has, in `Intl.PluralRules`' words.
 */
export const plural = (count: number, one: string, other = `${one}s`): string =>
  Math.round(count) === 1 ? one : other

/**
 * A count and the thing it counts, in the singular where there is one of it.
 *
 * The count is rounded before it is formatted and before the plural is chosen,
 * because a curve produces fractions and the user is shown how many cards.
 */
export const many = (count: number, one: string): string =>
  `${Math.round(count)} ${plural(count, one)}`

/** A whole number, grouped in thousands. */
export const groupDigits = (n: number): string => {
  const digits = String(Math.max(0, Math.floor(n)))
  let out = ''
  for (let i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 === 0) out += ' '
    out += digits[i]
  }
  return out
}
