/** The state a screen tells its files by, as this folder reads it. */

/**
 * The conflict a file a tab holds stands in, and nothing while it stands in
 * neither. A screen tells its files apart in its own words; these are the two
 * this folder has something to do about.
 */
export type FileConflict = 'gone' | 'stale' | null

/** Which conflict a screen's word names, and nothing for every other word. */
export const conflictIn = (state: string): FileConflict =>
  state === 'gone' || state === 'stale' ? state : null
