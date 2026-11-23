/** The enum value of each word a window uses, as the schema names it. Two values may not share a word. */
export const getCodesByWord = <W extends string, E extends number>(
  wordOf: Readonly<Record<E, W | null>>,
): Record<W, E> => {
  const pairs = Object.entries(wordOf).flatMap(([value, word]) =>
    word ? [[word, Number(value)] as const] : [],
  )
  const codes = Object.fromEntries(pairs) as Record<W, E>
  // The result is typed as holding every word, and nothing on the way here checks that.
  // Two values sharing a word leave one of them with no key, and reading it back gives undefined typed as the enum.
  if (Object.keys(codes).length !== pairs.length) {
    throw new Error(
      `two values of this enum answer to one word: ${pairs.map(([w]) => w).join(', ')}`,
    )
  }
  return codes
}
