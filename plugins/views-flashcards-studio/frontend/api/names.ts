/**
 * What a window calls each value of a wire enumeration, and what it sends back.
 *
 * The table of words is keyed by the enumeration, so the compiler asks for a word the moment the
 * enumeration carries a value more, and the sending side is read off that table.
 */

/** What each word a window uses is, as the wire enumeration names it. */
export const getNames = <W extends string, E extends string>(
  worded: Readonly<Record<E, W | null>>,
): Record<W, E> => {
  const named = Object.entries<W | null>(worded).flatMap(([value, word]) =>
    word ? [[word, value as E] as const] : [],
  )
  const reversed = Object.fromEntries(named) as Record<W, E>
  if (Object.keys(reversed).length !== named.length) {
    throw new Error(
      `two values of this enumeration answer to one word: ${named.map(([word]) => word).join(', ')}`,
    )
  }
  return reversed
}
