/**
 * The stylesheet the editor is rendered from.
 *
 * A rule the browser cannot parse is dropped silently, and so is a colour that
 * references a missing token. Both are checked here against the sheet the theme
 * emits.
 */
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { afterEach, describe, expect, it } from 'vitest'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { theme } from './theme'

const require = createRequire(import.meta.url)

/** The names the tokens declare, which is everything a stylesheet may read. */
const TOKENS = new Set(
  [
    ...readFileSync(require.resolve('@numen/ui/styles.css'), 'utf8').matchAll(
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

  // A selector the browser cannot parse drops its whole rule silently.
  it('is written in selectors a document can be requested for', () => {
    const content = selectors(getSheet())
    expect(content.length).toBeGreaterThan(20)

    for (const selector of content) {
      expect(() => document.querySelector(selector), selector).not.toThrow()
    }
  })

  // A colour is a token, and a name no token carries paints nothing.
  it('reads only tokens that are declared', () => {
    const readItem = [...getSheet().matchAll(/var\((--numen-[\w-]+)/g)].map(
      (found) => found[1] as string,
    )
    expect(readItem.length).toBeGreaterThan(10)

    for (const token of new Set(readItem)) {
      expect(TOKENS, token).toContain(token)
    }
  })
})
