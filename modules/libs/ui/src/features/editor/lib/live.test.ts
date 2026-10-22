import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { markdown } from '@codemirror/lang-markdown'
import { ensureSyntaxTree } from '@codemirror/language'
import { Compartment, EditorSelection, EditorState, StateEffect } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { GFM, type MarkdownConfig } from '@lezer/markdown'
import { live, wholeLines } from './live'
import { blockMarks, getMarks } from './marks'
import { Box, Bullet, Picture, Rule } from './widgets'
import { createState } from '../fixtures/state'

// Nothing here has a size, and the editor measures anyway.
Range.prototype.getClientRects = () =>
  Object.assign([], { item: () => null }) as unknown as DOMRectList
Range.prototype.getBoundingClientRect = () => new DOMRect()

interface Drawn {
  readonly from: number
  readonly to: number
  readonly spec: { class?: string; widget?: unknown }
}

/** Somewhere for the caret to stand that is in nothing. */
const ELSEWHERE = '\n\nelsewhere'

const collectDrawn = (state: EditorState): Drawn[] => {
  const found: Drawn[] = []
  const collect = (activeSet: ReturnType<typeof getMarks>) =>
    activeSet.between(
      0,
      state.doc.length,
      (from, to, deco) => void found.push({ from, to, spec: deco.spec ?? {} }),
    )

  collect(getMarks(state, 0, state.doc.length))
  collect(blockMarks(state))
  return found
}

const getDrawn = (doc: string, caret?: number): Drawn[] => {
  const text = caret === undefined ? doc + ELSEWHERE : doc
  return collectDrawn(createState(text, caret ?? text.length))
}

/** The stretch is drawn as something else, wherever the selection stands. */
const isHidden = (rendered: readonly Drawn[], from: number, to: number) =>
  rendered.some((deco) => deco.from === from && deco.to === to && !deco.spec.class)

/** A state whose selection is a caret at each of `carets`, and which is read-only if asked. */
const createCaretState = (doc: string, carets: number[], isReadOnly = false): EditorState => {
  const extra = [EditorState.allowMultipleSelections.of(true), EditorState.readOnly.of(isReadOnly)]
  return createState(doc)
    .update({ effects: StateEffect.appendConfig.of(extra) })
    .state.update({
      selection: EditorSelection.create(carets.map((at) => EditorSelection.cursor(at))),
    }).state
}

/** Whether the stretch of text is drawn as something else. */
const applyReplace = (doc: string, text: string, caret?: number) => {
  const pos = doc.indexOf(text)
  return getDrawn(doc, caret).some(
    (deco) => deco.from === pos && deco.to === pos + text.length && !deco.spec.class,
  )
}

const classes = (doc: string, caret?: number) =>
  getDrawn(doc, caret)
    .map((deco) => deco.spec.class)
    .filter((name): name is string => Boolean(name))

const widgets = (doc: string, caret?: number) =>
  getDrawn(doc, caret)
    .map((deco) => deco.spec.widget)
    .filter(Boolean)

describe('a heading', () => {
  it('is set by its level', () => {
    expect(classes('### Entropy')).toContain('cm-heading cm-heading-3')
  })

  it('hides its hashes and the space after them', () => {
    expect(applyReplace('### Entropy', '### ')).toBe(true)
  })

  it('shows them where the caret remains', () => {
    expect(applyReplace('### Entropy', '### ', 5)).toBe(false)
    expect(classes('### Entropy', 5)).toContain('cm-heading cm-heading-3')
  })
})

describe('marks around words', () => {
  it('hides what makes a word strong', () => {
    expect(classes('a **word** here')).toContain('cm-strong')
    expect(applyReplace('a **word** here', '**')).toBe(true)
  })

  it('brings them back under the caret', () => {
    expect(applyReplace('a **word** here', '**', 5)).toBe(false)
  })

  it('draws slanted and struck out text', () => {
    expect(classes('*a* ~~b~~')).toEqual(expect.arrayContaining(['cm-em', 'cm-strike']))
  })

  it('hides the backticks of inline code and keeps the code', () => {
    expect(classes('a `piece` of it')).toContain('cm-code-inline')
    expect(applyReplace('a `piece` of it', '`')).toBe(true)
  })
})

describe('a link', () => {
  it('is drawn as its words alone', () => {
    const doc = 'go [there](https://example.invalid)'
    expect(classes(doc)).toContain('cm-link')
    expect(applyReplace(doc, 'https://example.invalid')).toBe(true)
    expect(applyReplace(doc, '[')).toBe(true)
    expect(applyReplace(doc, ']')).toBe(true)
  })

  it('shows the address under the caret', () => {
    const doc = 'go [there](https://example.invalid)'
    expect(applyReplace(doc, 'https://example.invalid', 6)).toBe(false)
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
    const [waiting, isDone] = widgets('- [ ] a\n- [x] b\n').filter((it) => it instanceof Box)
    expect((waiting as Box).isTicked).toBe(false)
    expect((isDone as Box).isTicked).toBe(true)
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
    expect(applyReplace('> said\n', '> ')).toBe(true)
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
    const [deco] = getDrawn(TABLE).filter((it) => it.spec.widget)
    expect(deco?.from).toBe(0)
    expect(deco?.to).toBe(TABLE.trimEnd().length)
  })

  it('is shown as it is written where the caret remains in it', () => {
    expect(widgets(TABLE, 3)).toHaveLength(0)
  })
})

describe('the text itself', () => {
  it('is never changed by any of this', () => {
    const doc = '# a\n\n**b**\n\n| d | e |\n|---|---|\n| f | g |\n'
    expect(createState(doc).doc.toString()).toBe(doc)
  })
})

describe('what is shown as written', () => {
  it('is a construct any one of the carets remains in', () => {
    const doc = '**a** and **b**' + ELSEWHERE
    const drawnMarks = collectDrawn(createCaretState(doc, [3, 20]))
    expect(isHidden(drawnMarks, 0, 2)).toBe(false)
    expect(isHidden(drawnMarks, 3, 5)).toBe(false)
    expect(isHidden(drawnMarks, 10, 12)).toBe(true)
  })

  it('is a mark of quotation only under a caret in the quotation', () => {
    expect(applyReplace('> said\n', '> ', 3)).toBe(false)
    expect(applyReplace('> said\n', '> ')).toBe(true)
  })

  it('is the backticks of inline code under a caret in it', () => {
    expect(applyReplace('a `piece` of it', '`', 5)).toBe(false)
  })
})

describe('the space after a mark', () => {
  it('is hidden with the mark when there is one', () => {
    expect(isHidden(getDrawn('> said'), 0, 2)).toBe(true)
  })

  it('is not made up when there is none', () => {
    const drawnMarks = getDrawn('>said')
    expect(isHidden(drawnMarks, 0, 1)).toBe(true)
    expect(isHidden(drawnMarks, 0, 2)).toBe(false)
  })
})

describe('what stays as written', () => {
  it('is the line under a heading', () => {
    const doc = 'Title\n===\n'
    expect(classes(doc)).toContain('cm-heading cm-heading-1')
    expect(applyReplace(doc, '===')).toBe(false)
  })

  it('is the fence of a block of code', () => {
    expect(applyReplace('```go\nx := 1\n```\n', '```')).toBe(false)
  })

  it('is text that no construct remains over', () => {
    expect(getDrawn('plain words')).toEqual([])
  })

  it('is the words of a picture, and what is inside them, under the caret', () => {
    expect(classes('![a **b**](p.png)', 4)).toContain('cm-strong')
  })

  it('is the address of a picture, which has its own drawing', () => {
    expect(getDrawn('![a](p.png)')).toHaveLength(1)
  })
})

describe('a box for something to do', () => {
  it('can be ticked where the text can be written', () => {
    const [box] = collectDrawn(createCaretState('- [ ] a' + ELSEWHERE, [0]))
      .map((deco) => deco.spec.widget)
      .filter((it): it is Box => it instanceof Box)
    expect(box?.isWritable).toBe(true)
  })

  it('cannot where the text is read-only', () => {
    const [box] = collectDrawn(createCaretState('- [ ] a' + ELSEWHERE, [0], true))
      .map((deco) => deco.spec.widget)
      .filter((it): it is Box => it instanceof Box)
    expect(box?.isWritable).toBe(false)
  })
})

describe('whole lines', () => {
  const TABLE = '| a | **b** |\n|---|---|\n| c | d |\n'

  it('are drawn as blocks', () => {
    const blocks = getDrawn('a\n\n---\n\nb\n').filter((deco) => deco.spec.widget)
    expect(blocks).toHaveLength(1)
    expect((blocks[0]?.spec as { block?: boolean }).block).toBe(true)
  })

  it('leave the words inside a table as written when it is drawn as a table', () => {
    expect(classes(TABLE)).not.toContain('cm-strong')
  })

  it('give the words inside a table their marks where the caret remains in it', () => {
    expect(classes(TABLE, 3)).toContain('cm-strong')
  })
})

/** A state with a parser that can be swapped, and the table and rule widgets held in it. */
const createLinesState = (doc: string, caret = doc.length) => {
  const language = new Compartment()
  const state = EditorState.create({
    doc,
    selection: { anchor: caret },
    extensions: [
      language.of(markdown({ extensions: GFM })),
      EditorState.allowMultipleSelections.of(true),
      wholeLines,
    ],
  })
  ensureSyntaxTree(state, doc.length, 5000)
  return { state, language }
}

interface Standing {
  readonly from: number
  readonly to: number
  readonly deco: { spec: { widget?: unknown; block?: boolean } }
}

const getStanding = (state: EditorState): Standing[] => {
  const found: Standing[] = []
  state
    .field(wholeLines)
    .marks.between(0, state.doc.length, (from, to, deco) => void found.push({ from, to, deco }))
  return found
}

describe('the marks of whole lines retained in the state', () => {
  const RULE = 'a\n\n---\n\nb\n\nelse'
  const INTRO = 'intro\n\n| a | b |\n|---|---|\n| c | d |\n\nelse'

  it('carry on unchanged when text is typed before a rule', () => {
    const { state } = createLinesState(RULE)
    const [was] = getStanding(state)
    const [now] = getStanding(state.update({ changes: { from: 0, insert: 'xy' } }).state)
    expect(now?.deco).toBe(was?.deco)
    expect([now?.from, now?.to]).toEqual([was ? was.from + 2 : -1, was ? was.to + 2 : -1])
  })

  it('carry on unchanged when text is typed after a rule', () => {
    const { state } = createLinesState(RULE)
    const [was] = getStanding(state)
    const [now] = getStanding(state.update({ changes: { from: RULE.length, insert: 'xy' } }).state)
    expect(now?.deco).toBe(was?.deco)
  })

  it('are drawn again when text is typed at the first character of a rule', () => {
    const { state } = createLinesState(RULE)
    const [was] = getStanding(state)
    const [now] = getStanding(
      state.update({ changes: { from: was?.from ?? 0, insert: '\n' } }).state,
    )
    expect(now?.deco.spec.widget).toBeInstanceOf(Rule)
    expect(now?.deco.spec.widget).not.toBe(was?.deco.spec.widget)
  })

  it('move a table with the text before it', () => {
    const { state } = createLinesState(INTRO)
    const [was] = getStanding(state)
    const [now] = getStanding(state.update({ changes: { from: 0, insert: 'x' } }).state)
    const table = (now?.deco.spec.widget as { table: { from: number } }).table
    expect(table.from).toBe((was?.from ?? 0) + 1)
    expect(now?.from).toBe((was?.from ?? 0) + 1)
    expect(now?.deco.spec.block).toBe(true)
  })

  it('show a rule as written when a second caret remains in it', () => {
    const { state } = createLinesState(RULE)
    expect(getStanding(state)).toHaveLength(1)

    const moved = state.update({
      selection: EditorSelection.create([
        EditorSelection.cursor(RULE.length),
        EditorSelection.cursor(4),
      ]),
    }).state
    expect(getStanding(moved)).toHaveLength(0)
  })

  it('keep their marks and take the nodes of the new parse when the same blocks are found', () => {
    const { state, language } = createLinesState(RULE)
    const was = state.field(wholeLines)

    const swapped = state.update({ effects: language.reconfigure(markdown()) }).state
    const now = swapped.field(wholeLines)
    expect(now.marks).toBe(was.marks)
    expect(now.nodes).toEqual([{ name: 'HorizontalRule', from: 3, to: 6 }])
    expect(now.nodes).not.toBe(was.nodes)
  })

  it('follow the parser when it is swapped without the text changing', () => {
    const { state, language } = createLinesState(INTRO)
    expect(getStanding(state)).toHaveLength(1)

    const swapped = state.update({ effects: language.reconfigure(markdown()) }).state
    expect(getStanding(swapped)).toHaveLength(0)
  })
})

/** A parser that reads the consecutive lines starting with `lead` as one rule, inset at either end. */
const readRulesAt = (lead: string, insetFrom = 0, insetTo = 0): MarkdownConfig => ({
  parseBlock: [
    {
      name: 'LeadRule',
      before: 'HorizontalRule',
      parse: (cx, line) => {
        if (!line.text.startsWith(lead)) return false
        const from = cx.lineStart
        while (cx.nextLine() && line.text.startsWith(lead)) continue
        cx.addElement(cx.elt('HorizontalRule', from + insetFrom, cx.prevLineEnd() - insetTo))
        return true
      },
    },
  ],
})

describe('the marks of whole lines when the parser finds other blocks in the same text', () => {
  const swapTo = (doc: string, config: MarkdownConfig) => {
    const { state, language } = createLinesState(doc)
    const swapped = state.update({
      effects: language.reconfigure(markdown({ extensions: [GFM, config] })),
    }).state
    return {
      was: getStanding(state),
      now: getStanding(swapped),
      nodes: swapped.field(wholeLines).nodes,
    }
  }

  it('are drawn again when a table is found to be a rule', () => {
    const { was, now } = swapTo('| a | b |\n|---|---|\n| c | d |\n\nelse', readRulesAt('|'))
    expect(was[0]?.deco.spec.widget).not.toBeInstanceOf(Rule)
    expect(now).toHaveLength(1)
    expect(now[0]?.deco.spec.widget).toBeInstanceOf(Rule)
  })

  it('are drawn again when a rule is found to start later', () => {
    const { was, now, nodes } = swapTo('a\n\n---\n\nb\n\nelse', readRulesAt('-', 1))
    expect(now[0]?.deco).not.toBe(was[0]?.deco)
    expect(nodes).toEqual([{ name: 'HorizontalRule', from: 4, to: 6 }])
  })

  it('are drawn again when a rule is found to end earlier', () => {
    const { was, now, nodes } = swapTo('a\n\n---\n\nb\n\nelse', readRulesAt('-', 0, 1))
    expect(now[0]?.deco).not.toBe(was[0]?.deco)
    expect(nodes).toEqual([{ name: 'HorizontalRule', from: 3, to: 5 }])
  })

  it('are drawn again when a change far away makes a rule of what was code', () => {
    const doc = '---\n\n```\np\n\n---\n\n```\n\nz\n'
    const { state } = createLinesState(doc)
    const [was] = getStanding(state)

    const [now, ...others] = getStanding(state.update({ changes: { from: 0, to: 9 } }).state)
    expect(others).toHaveLength(0)
    expect([now?.from, now?.to]).toEqual([3, 6])
    expect(now?.deco.spec.widget).toBeInstanceOf(Rule)
    expect(now?.deco.spec.widget).not.toBe(was?.deco.spec.widget)
  })
})

describe('the marks drawn for the lines on screen', () => {
  // Eight characters to a line, so 2000 of margin is a line start.
  const DOC = '**a** b\n'.repeat(1000)
  const PING = StateEffect.define<null>()
  let visible: { from: number; to: number }[] = []

  beforeEach(() => {
    visible = []
    vi.spyOn(EditorView.prototype, 'visibleRanges', 'get').mockImplementation(() => visible)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  const open = (doc: string, ranges: { from: number; to: number }[]) => {
    visible = ranges
    const language = new Compartment()
    const view = new EditorView({
      parent: document.body,
      state: EditorState.create({
        doc,
        extensions: [language.of(markdown({ extensions: GFM })), live],
      }),
    })
    ensureSyntaxTree(view.state, doc.length, 5000)
    // The tree parsed to the end joins the state with the first transaction.
    view.dispatch({ effects: PING.of(null) })
    return { view, language }
  }

  const getPlugin = (view: EditorView) => {
    const plugin = view.plugin(live)
    if (!plugin) throw new Error('the plugin is not in the view')
    return plugin
  }

  /** Where each strong text starts among the marks in hand. */
  const getStrongs = (view: EditorView) => {
    const found: number[] = []
    getPlugin(view).decorations.between(0, view.state.doc.length, (from, _to, deco) => {
      if ((deco.spec as { class?: string }).class === 'cm-strong') found.push(from)
    })
    return found
  }

  const isDrawnAt = (view: EditorView, from: number, to: number) => {
    let concealed = false
    getPlugin(view).decorations.between(from, to, (start, end, deco) => {
      if (start === from && end === to && !(deco.spec as { class?: string }).class) concealed = true
    })
    return concealed
  }

  it('reach a margin past the lines on screen, and no further', () => {
    const { view } = open(DOC, [{ from: 4003, to: 4005 }])
    const strongs = getStrongs(view)
    expect(strongs[0]).toBe(2000)
    expect(strongs.at(-1)).toBe(6000)
    view.destroy()
  })

  it('reach from the last of the visible ranges', () => {
    const { view } = open(DOC, [
      { from: 4003, to: 4005 },
      { from: 4403, to: 4405 },
    ])
    const strongs = getStrongs(view)
    expect(strongs[0]).toBe(2000)
    expect(strongs.at(-1)).toBe(6400)
    view.destroy()
  })

  it('start at the top of the text when the screen is near it', () => {
    const { view } = open(DOC, [{ from: 10, to: 12 }])
    const strongs = getStrongs(view)
    expect(strongs[0]).toBe(0)
    expect(strongs.at(-1)).toBe(2008)
    view.destroy()
  })

  it('stop at the bottom of the text when the screen is near it', () => {
    const { view } = open(DOC, [{ from: 7990, to: 7995 }])
    const strongs = getStrongs(view)
    expect(strongs[0]).toBe(5984)
    expect(strongs.at(-1)).toBe(7992)
    view.destroy()
  })

  it('are places the caret cannot be put', () => {
    const { view } = open(DOC, [{ from: 4003, to: 4005 }])
    const sizes = view.state.facet(EditorView.atomicRanges).map((get) => get(view).size)
    expect(sizes.some((size) => size > 0)).toBe(true)
    view.destroy()
  })

  it('are no places the caret cannot be put once the plugin has crashed', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    vi.spyOn(EditorView.prototype, 'visibleRanges', 'get').mockImplementation(() => {
      throw new Error('no ranges')
    })
    const { view } = open(DOC, [])
    const sizes = view.state.facet(EditorView.atomicRanges).map((get) => get(view).size)
    expect(sizes.every((size) => size === 0)).toBe(true)
    view.destroy()
  })

  it('are none while nothing is on screen', () => {
    const { view } = open(DOC, [])
    expect(getPlugin(view).decorations.size).toBe(0)
    expect(getPlugin(view).span).toBeNull()
    view.destroy()
  })

  it('are drawn once the screen has something on it', () => {
    const { view } = open(DOC, [])
    visible = [{ from: 4003, to: 4005 }]
    view.dispatch({ effects: PING.of(null) })
    expect(getStrongs(view)[0]).toBe(2000)
    view.destroy()
  })

  it('are dropped once the screen has nothing on it', () => {
    const { view } = open(DOC, [{ from: 4003, to: 4005 }])
    visible = []
    view.dispatch({ effects: PING.of(null) })
    expect(getPlugin(view).decorations.size).toBe(0)
    view.destroy()
  })

  describe.each([
    ['inside the span', [{ from: 3500, to: 3510 }]],
    ['at the start of the span', [{ from: 2000, to: 2003 }]],
    ['at the end of the span', [{ from: 6000, to: 6003 }]],
  ])('a screen %s', (_name, ranges) => {
    it('keeps the marks it has', () => {
      const { view } = open(DOC, [{ from: 4003, to: 4005 }])
      const was = getPlugin(view).decorations
      visible = ranges
      view.dispatch({ effects: PING.of(null) })
      expect(getPlugin(view).decorations).toBe(was)
      view.destroy()
    })
  })

  it('are drawn again for a screen that moves above the span', () => {
    const { view } = open(DOC, [{ from: 4003, to: 4005 }])
    const was = getPlugin(view).decorations
    visible = [{ from: 1000, to: 1003 }]
    view.dispatch({ effects: PING.of(null) })
    expect(getPlugin(view).decorations).not.toBe(was)
    const strongs = getStrongs(view)
    expect(strongs[0]).toBe(0)
    expect(strongs.at(-1)).toBe(3000)
    view.destroy()
  })

  it('are drawn again for a screen that moves below the span', () => {
    const { view } = open(DOC, [{ from: 4003, to: 4005 }])
    const was = getPlugin(view).decorations
    visible = [{ from: 6008, to: 6010 }]
    view.dispatch({ effects: PING.of(null) })
    expect(getPlugin(view).decorations).not.toBe(was)
    const strongs = getStrongs(view)
    expect(strongs[0]).toBe(4008)
    expect(strongs.at(-1)).toBe(7992)
    view.destroy()
  })

  it('are drawn again where the text is typed in', () => {
    const { view } = open(DOC, [{ from: 4003, to: 4005 }])
    view.dispatch({ changes: { from: 0, insert: 'z' } })
    const strongs = getStrongs(view)
    expect(strongs).toContain(4001)
    expect(strongs).not.toContain(4000)
    view.destroy()
  })

  it('are drawn again where the caret moves into a construct', () => {
    const { view } = open(DOC, [{ from: 4003, to: 4005 }])
    expect(isDrawnAt(view, 4000, 4002)).toBe(true)

    view.dispatch({ selection: { anchor: 4001 } })
    expect(isDrawnAt(view, 4000, 4002)).toBe(false)
    view.destroy()
  })

  it('are drawn again where the parser is swapped without the text changing', () => {
    const doc = '~~a~~ b\n'
    const { view, language } = open(doc, [{ from: 0, to: 3 }])
    const struckVal = () => {
      let found = false
      getPlugin(view).decorations.between(0, doc.length, (_f, _t, deco) => {
        if ((deco.spec as { class?: string }).class === 'cm-strike') found = true
      })
      return found
    }
    expect(struckVal()).toBe(true)

    view.dispatch({ effects: language.reconfigure(markdown()) })
    expect(struckVal()).toBe(false)
    view.destroy()
  })
})
