/** Texts of a note tab whose saving stopped because its file changed on disk. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  stale: 'Saving paused: the file changed on disk.',
  gone: 'Saving paused: the note was removed from the vault. Your text is kept.',
  keep: 'Keep my version',
  take: 'Use disk version',
  makeAgain: 'Recreate it',
}
