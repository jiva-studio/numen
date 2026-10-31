/**
 * What a window calls each value of a schema enum, and what it sends back.
 *
 * The table of words is keyed by the enum the schema generates, so the compiler
 * requires a word as soon as the schema gains a value, and the sending side is
 * derived from that table.
 */

/** What each word a window uses is, as the schema names it. */
export const getNames = <W extends string, E extends number>(
  worded: Readonly<Record<E, W | null>>,
): Record<W, E> => {
  const named = Object.entries(worded).flatMap(([value, word]) =>
    word ? [[word, Number(value)] as const] : [],
  )
  const reversed = Object.fromEntries(named) as Record<W, E>
  // The reverse table is typed as holding every word, and nothing on the way
  // here checks that. Two values sharing a word leave one of them with no key,
  // and reading it back gives undefined wearing the enum's type.
  if (Object.keys(reversed).length !== named.length) {
    throw new Error(
      `two values of this enum answer to one word: ${named.map(([w]) => w).join(', ')}`,
    )
  }
  return reversed
}
