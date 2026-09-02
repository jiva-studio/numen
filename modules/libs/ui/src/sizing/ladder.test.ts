/**
 * Every size text is set at, against the ladder the tokens name.
 *
 * A component names a size; it never writes one.
 */
import { describe, expect, it } from 'vitest'

/**
 * The module's source, as text. The tokens' own file declares the ladder and
 * the root's font size, and a story draws a length on purpose.
 */
const SOURCE = import.meta.glob<string>('../**/*.{vue,css,ts}', {
  query: '?raw',
  import: 'default',
  eager: true,
})

const specimen = (path: string) => path.endsWith('.stories.ts') || path.endsWith('.test.ts')

/** Where a size is set: in a rule, and in the style a component hands a widget. */
const IN_CSS = /(?<![-\w])font-size\s*:\s*([^;}"'`]+)/g
const IN_JS = /(?<![-\w])fontSize\s*:\s*'([^']*)'/g

/** A size is a token by name, the size around it, or a multiple of that. */
const LADDER = /^(inherit|[\d.]+em|var\(--[\w-]+\))$/

/** Every size set in one file, in the words it is written in. */
const sizes = (text: string): string[] =>
  [...text.matchAll(IN_CSS), ...text.matchAll(IN_JS)].map((found) => (found[1] ?? '').trim())

describe('the type ladder', () => {
  it('is what every size text is set at is named by', () => {
    const written: string[] = []
    for (const [path, text] of Object.entries(SOURCE)) {
      if (path.includes('/tokens/') || specimen(path)) continue
      for (const size of sizes(text)) {
        if (!LADDER.test(size)) written.push(`${path}: ${size}`)
      }
    }
    expect(written).toEqual([])
  })
})
