/**
 * The two halves of a character outside the basic plane, which is two UTF-16
 * code units.
 */

/** Whether the code unit is the leading half of a pair. */
export const isLeading = (unit: number): boolean => unit >= 0xd800 && unit <= 0xdbff

/** Whether the code unit is the trailing half of a pair. */
export const isTrailing = (unit: number): boolean => unit >= 0xdc00 && unit <= 0xdfff
