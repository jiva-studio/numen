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
 * The thing a count counts, in the singular where there is one of it.
 *
 * Apart from `many` because a layout that puts the figure and the noun in
 * fields of their own still has the noun to make plural.
 *
 * `other` is the whole plural where an `s` on the end does not give it: an
 * irregular noun, or a phrase whose plural falls inside it. `one` and `other`
 * are the two forms English has, in the words `Intl.PluralRules` gives them.
 */
export const plural = (count: number, one: string, other = `${one}s`): string =>
  Math.round(count) === 1 ? one : other

/**
 * A count and the thing it counts, in the singular where there is one of it.
 *
 * The count is rounded before it is read and before it is made plural, because
 * a curve answers in fractions and a person is being told how many cards.
 */
export const many = (count: number, one: string): string =>
  `${Math.round(count)} ${plural(count, one)}`

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
