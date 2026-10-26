/** What the tab holding the settings file says. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  /** What the tab is called, which is the file it holds. */
  called: 'numen.json',
  /** The file could not be read. */
  unread: 'File unreadable:',
  /** What was typed is not settings this build can read. */
  unwritten: 'File not saved:',
  /** Shown in place of the editor until the file is read. */
  loading: 'Loading…',
  /** The file moved past what this tab read, and the two ways out. */
  stale: 'Saving paused: the file changed on disk.',
  keep: 'Keep my version',
  take: 'Use disk version',
}
