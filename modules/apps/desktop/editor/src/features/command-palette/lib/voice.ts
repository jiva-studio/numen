/**
 * How several things a command has to say become the one line the window says
 * them in.
 */

/** What a command did to other notes, named once each under what it did. */
export const formatNames = (says: string, notes: readonly string[]): string => {
  const named = [...new Set(notes)]
  return named.length === 0 ? '' : `${says} ${named.join(', ')}`
}

/** Everything one command has to say, as the one line the window says it in. */
export const all = (...says: readonly string[]): string => says.filter((one) => one).join('. ')
