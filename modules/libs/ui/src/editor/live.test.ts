import { describe, expect, it } from 'vitest'
import type { EditorState } from '@codemirror/state'
import { blockMarks, marks } from './live'
import { Box, Bullet, Picture, Rule } from './widgets'
import { parsed } from '@/fixtures/parsed'

interface Drawn {
  readonly from: number
  readonly to: number
  readonly spec: { class?: string; widget?: unknown }
}

/** Somewhere for the caret to stand that is in nothing. */
const ELSEWHERE = '\n\nelsewhere'

const drawn = (doc: string, caret?: number): Drawn[] => {
  const text = caret === undefined ? doc + ELSEWHERE : doc
  const state: EditorState = parsed(text, caret ?? text.length)
  const found: Drawn[] = []
  const collect = (set: ReturnType<typeof marks>) =>
    set.between(0, state.doc.length, (from, to, deco) =>
      void found.push({ from, to, spec: deco.spec ?? {} }),
    )

  collect(marks(state, 0, state.doc.length))
  collect(blockMarks(state))
  return found
}

/** Whether the stretch of text is drawn as something else. */
const replaced = (doc: string, text: string, caret?: number) => {
  const at = doc.indexOf(text)
  return drawn(doc, caret).some(
    (deco) => deco.from === at && deco.to === at + text.length && !deco.spec.class,
  )
}

const classes = (doc: string, caret?: number) =>
  drawn(doc, caret)
    .map((deco) => deco.spec.class)
    .filter((name): name is string => Boolean(name))

const widgets = (doc: string, caret?: number) =>
  drawn(doc, caret)
    .map((deco) => deco.spec.widget)
    .filter(Boolean)

describe('a heading', () => {
  it('is set by its level', () => {
    expect(classes('### Entropy')).toContain('cm-heading cm-heading-3')
  })

  it('hides its hashes and the space after them', () => {
    expect(replaced('### Entropy', '### ')).toBe(true)
  })

  it('shows them where the caret stands', () => {
    expect(replaced('### Entropy', '### ', 5)).toBe(false)
    expect(classes('### Entropy', 5)).toContain('cm-heading cm-heading-3')
  })
})

describe('marks around words', () => {
  it('hides what makes a word strong', () => {
    expect(classes('a **word** here')).toContain('cm-strong')
    expect(replaced('a **word** here', '**')).toBe(true)
  })

  it('brings them back under the caret', () => {
    expect(replaced('a **word** here', '**', 5)).toBe(false)
  })

  it('draws slanted and struck out text', () => {
    expect(classes('*a* ~~b~~')).toEqual(expect.arrayContaining(['cm-em', 'cm-strike']))
  })

  it('hides the backticks of inline code and keeps the code', () => {
    expect(classes('a `piece` of it')).toContain('cm-code-inline')
    expect(replaced('a `piece` of it', '`')).toBe(true)
  })
})

describe('a link', () => {
  it('is drawn as its words alone', () => {
    const doc = 'go [there](https://example.invalid)'
    expect(classes(doc)).toContain('cm-link')
    expect(replaced(doc, 'https://example.invalid')).toBe(true)
    expect(replaced(doc, '[')).toBe(true)
    expect(replaced(doc, ']')).toBe(true)
  })

  it('shows the address under the caret', () => {
    const doc = 'go [there](https://example.invalid)'
    expect(replaced(doc, 'https://example.invalid', 6)).toBe(false)
  })
})

describe('a list', () => {
  it('draws a disc for a bullet', () => {
    expect(widgets('- a\n- b\n')[0]).toBeInstanceOf(Bullet)
  })

  it('leaves the number of an ordered list alone', () => {
    expect(widgets('1. a\n2. b\n')).toHaveLength(0)
  })

  it('draws a box for something to do, ticked or not', () => {
    const [waiting, done] = widgets('- [ ] a\n- [x] b\n').filter((it) => it instanceof Box)
    expect((waiting as Box).done).toBe(false)
    expect((done as Box).done).toBe(true)
  })
})

describe('blocks', () => {
  it('draws a line for a rule, and shows the dashes under the caret', () => {
    expect(widgets('a\n\n---\n\nb\n')[0]).toBeInstanceOf(Rule)
    expect(widgets('a\n\n---\n\nb\n', 4)).toHaveLength(0)
  })

  it('gives every line of a fenced block the same ground', () => {
    const drawnClasses = classes('```go\nx := 1\n```\n')
    expect(drawnClasses).toContain('cm-code cm-code-first')
    expect(drawnClasses).toContain('cm-code cm-code-last')
    expect(drawnClasses.filter((name) => name.startsWith('cm-code'))).toHaveLength(3)
  })

  it('marks the lines of a quotation and hides its marks', () => {
    expect(classes('> said\n> and said\n')).toEqual(['cm-quote', 'cm-quote'])
    expect(replaced('> said\n', '> ')).toBe(true)
  })
})

describe('a picture', () => {
  it('is drawn where its address was written', () => {
    const widget = widgets('![a](p.png)')[0]
    expect(widget).toBeInstanceOf(Picture)
    expect((widget as Picture).address).toBe('p.png')
  })

  it('is written out again under the caret', () => {
    expect(widgets('![a](p.png)', 3)).toHaveLength(0)
  })
})

describe('a table', () => {
  const TABLE = '| a | b |\n|---|---|\n| c | d |\n'

  it('is drawn as a table', () => {
    expect(widgets(TABLE)).toHaveLength(1)
  })

  it('takes whole lines', () => {
    const [deco] = drawn(TABLE).filter((it) => it.spec.widget)
    expect(deco?.from).toBe(0)
    expect(deco?.to).toBe(TABLE.trimEnd().length)
  })

  it('is shown as it is written where the caret stands in it', () => {
    expect(widgets(TABLE, 3)).toHaveLength(0)
  })
})

describe('the text itself', () => {
  it('is never changed by any of this', () => {
    const doc = '# a\n\n**b**\n\n| d | e |\n|---|---|\n| f | g |\n'
    expect(parsed(doc).doc.toString()).toBe(doc)
  })
})
