/** Combines the messages of a command into the single line the window shows. */

/** What a command did to other notes, named once each under what it did. */
export const formatNames = (says: string, notes: readonly string[]): string => {
  const named = [...new Set(notes)]
  return named.length === 0 ? '' : `${says} ${named.join(', ')}`
}

/** Joins all messages of one command into a single line. */
export const joinAll = (...says: readonly string[]): string => says.filter((one) => one).join('. ')
