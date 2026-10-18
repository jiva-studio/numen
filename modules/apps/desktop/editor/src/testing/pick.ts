/** The entry at a place of a list, which a test needs there: a missing one fails the test. */
export const pick = <T>(list: readonly T[], at: number): T => {
  const found = list[at]
  if (found === undefined) throw new Error(`nothing at ${at} of ${list.length}`)
  return found
}
