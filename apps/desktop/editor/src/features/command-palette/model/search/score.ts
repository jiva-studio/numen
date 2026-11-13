/**
 * Search groups and group ranking limits.
 */
/** Which group is which, and nothing else is one. */
export type SearchGroup = 'names' | 'text' | 'meaning'

/** How many answers each group holds. */
export const GROUP_RESULT_LIMIT = 8
