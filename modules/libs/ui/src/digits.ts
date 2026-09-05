/**
 * Numbers as they are read out.
 *
 * Here rather than in a component because a count of text reaches six figures on
 * an ordinary vault, and the grouping is the same wherever it is drawn.
 */

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
