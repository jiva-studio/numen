/** The enumeration value of each word a window uses, as the wire names it. Two values may not share a word. */
export const getCodesByWord = <W extends string, E extends string>(
  wordOf: Readonly<Record<E, W | null>>,
): Record<W, E> => {
  const pairs = Object.entries<W | null>(wordOf).flatMap(([value, word]) =>
    word ? [[word, value as E] as const] : [],
  )
  const codes = Object.fromEntries(pairs) as Record<W, E>
  if (Object.keys(codes).length !== pairs.length) {
    throw new Error(
      `two values of this enumeration answer to one word: ${pairs.map(([word]) => word).join(', ')}`,
    )
  }
  return codes
}
