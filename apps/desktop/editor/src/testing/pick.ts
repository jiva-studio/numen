/** The entry at a place of a list, which a test needs there: a missing one fails the test. */
export const pick = <T>(list: readonly T[], pos: number): T => {
  const found = list[pos]
  if (found === undefined) throw new Error(`nothing at ${pos} of ${list.length}`)
  return found
}
