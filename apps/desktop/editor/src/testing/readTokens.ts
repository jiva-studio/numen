/**
 * The declarations a piece of CSS holds, by property name: the text before each colon and the text
 * after it up to the next semicolon or brace. A property that leaks out of the rule it was
 * written for shows up here as one more name.
 */
export const readTokens = (css: string): ReadonlyMap<string, string> =>
  new Map(
    Array.from(css.matchAll(/([-\w]+)\s*:\s*([^;{}]+)/g), (found) => [
      found[1] ?? '',
      (found[2] ?? '').trim(),
    ]),
  )
