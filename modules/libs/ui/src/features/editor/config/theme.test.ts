/**
 * The stylesheet the editor is painted from.
 *
 * A rule the browser cannot read is dropped where nobody is told, and so is a
 * colour named after a token that is not there. Both are read back here off the
 * sheet the theme actually lays down.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { afterEach, describe, expect, it } from 'vitest'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { theme } from './theme'

/** The names the tokens declare, which is everything a stylesheet may read. */
const TOKENS = new Set(
  [
    ...readFileSync(join(process.cwd(), 'src/shared/tokens/tokens.css'), 'utf8').matchAll(
      /(--numen-[\w-]+)\s*:/g,
    ),
  ].map((found) => found[1] as string),
)

let mounted: EditorView | null = null

afterEach(() => {
  mounted?.destroy()
  mounted = null
})

/** The sheet the theme lays into the page once a view is painted with it. */
function getSheet(): string {
  mounted = new EditorView({ state: EditorState.create({ extensions: [theme] }) })
  const sheets = [...document.head.querySelectorAll('style')]
    .map((one) => one.textContent ?? '')
    .filter((css) => css.includes('--editor-mark'))
  return sheets.join('\n')
}

/** Everything standing before a block of declarations. */
const selectors = (css: string): readonly string[] =>
  [...css.matchAll(/(?:^|[}\n])\s*([^{}@\n][^{}]*?)\s*\{/g)].map((found) =>
    (found[1] as string).replace(/\s+/g, ' ').trim(),
  )

describe('the editor’s stylesheet', () => {
  it('is laid into the page when a view is painted with it', () => {
    expect(getSheet()).not.toBe('')
  })

  // A selector the browser cannot read takes its whole rule with it, and says
  // nothing about it.
  it('is written in selectors a document can be asked for', () => {
    const written = selectors(getSheet())
    expect(written.length).toBeGreaterThan(20)

    for (const selector of written) {
      expect(() => document.querySelector(selector), selector).not.toThrow()
    }
  })

  // A colour is a token, and a name no token carries paints nothing.
  it('reads only tokens that are declared', () => {
    const read = [...getSheet().matchAll(/var\((--numen-[\w-]+)/g)].map(
      (found) => found[1] as string,
    )
    expect(read.length).toBeGreaterThan(10)

    for (const token of new Set(read)) {
      expect(TOKENS, token).toContain(token)
    }
  })
})
