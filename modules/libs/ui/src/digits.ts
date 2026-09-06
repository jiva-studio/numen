/**
 * Numbers as they are read out.
 *
 * Here rather than in a component because a count of text reaches six figures on
 * an ordinary vault, and the grouping is the same wherever it is drawn.
 */

/**
 * A share as a person reads one, which is a percentage and not a fraction.
 *
 * Rounded, so a half is read as fifty and a whole as a hundred. A count that
 * has still to finish reads by `percentWord`, which never rounds up to a
 * hundred.
 */
export const percent = (share: number): string => `${Math.round(share * 100)}%`

/**
 * A count and the thing it counts, in the singular where there is one of it.
 *
 * The count is rounded before it is read and before it is made plural, because
 * a curve answers in fractions and a person is being told how many cards.
 */
export const many = (count: number, one: string): string => {
  const whole = Math.round(count)
  return `${whole} ${whole === 1 ? one : `${one}s`}`
}

/** A whole number, grouped in thousands. */
export const grouped = (n: number): string => {
  const digits = String(Math.max(0, Math.floor(n)))
  let out = ''
  for (let i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 === 0) out += ' '
    out += digits[i]
  }
  return out
}
