/** What a note tab says: the questions its file puts, and the two ways out. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  stale: 'Saving paused: the file changed on disk.',
  gone: 'Saving paused: the note was removed from the vault. Your text is kept.',
  makeAgain: 'recreate it',
  keep: 'Keep my version',
  take: 'Use disk version',
}
