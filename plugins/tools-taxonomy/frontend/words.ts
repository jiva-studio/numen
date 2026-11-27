// Stryker disable StringLiteral: interface copy
/** Words used across the taxonomy frontend. */

/** The count of the tags a row of the file tree does not draw a dot for. */
export const getRestText = (count: number): string => `+${count}`

/** The title of the list the command Set tags opens. */
export const TAGS_TITLE = 'Tags'

/** The placeholder of the field of that list. */
export const TAG_PLACEHOLDER = 'Tag'

/** The title of the row that makes a tag of the text typed. */
export const getCreateTagTitle = (text: string): string => `Create ${text}`

/** The detail of a tag the command Delete tag lists: the number of notes it is removed from. */
export const getDeleteTagDetail = (count: number): string =>
  count === 1 ? '1 note' : `${count} notes`
