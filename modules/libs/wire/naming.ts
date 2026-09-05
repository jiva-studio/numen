/**
 * What a window calls each value of a schema enum, and what it sends back.
 *
 * A table of words is keyed by the enum the schema generates, so the compiler
 * asks for a word the moment the schema carries a value more. A table keyed the
 * other way — by the words themselves — is a second copy of the enum, and a
 * value added to the schema passes it in silence. So the sending side is read
 * off the table of words rather than written out again.
 */

/** What each word a window uses is, as the schema names it. */
export const namesOf = <W extends string, E extends number>(
  worded: Readonly<Record<E, W | null>>,
): Record<W, E> =>
  Object.fromEntries(
    Object.entries(worded).flatMap(([value, word]) => (word ? [[word, Number(value)]] : [])),
  ) as Record<W, E>
