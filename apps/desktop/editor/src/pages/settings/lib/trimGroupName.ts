/** The characters Go's `unicode.IsSpace` accepts, which `strings.TrimSpace` removes at the ends. */
const SPACE =
  '\\t\\n\\v\\f\\r \\u0085\\u00A0\\u1680\\u2000-\\u200A\\u2028\\u2029\\u202F\\u205F\\u3000'
const ENDS = new RegExp(`^[${SPACE}]+|[${SPACE}]+$`, 'gu')

/** Returns the name without the space at its ends, as the core trims a group name. */
export function trimGroupName(name: string): string {
  return name.replace(ENDS, '')
}

/** The longest name of a group, in characters; the core refuses a longer one. */
export const MAX_GROUP_LENGTH = 64
