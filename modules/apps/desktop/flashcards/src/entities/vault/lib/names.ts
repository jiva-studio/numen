/** The name of a deck, as it is shown: the file's, without folders or suffix. */
export const deckName = (path: string): string => {
  const last = path.split('/').pop() ?? path
  return last.replace(/\.[^.]+$/, '')
}
