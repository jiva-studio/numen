import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { markdown } from '@codemirror/lang-markdown'
import { ensureSyntaxTree } from '@codemirror/language'
import { EditorState, type ChangeSpec } from '@codemirror/state'
import { GFM } from '@lezer/markdown'
import { LANGUAGES } from '../../lib/config/languages'
import { wholeLines } from './live'
import { blockMarks } from './marks'
import { Rule } from './widgets'

const TABLE = '| a | b |\n|---|---|\n| 1 | 2 |'
const OTHER = '| c | d |\n|---|---|\n| 3 | 4 |'

const create = (doc: string, caret = doc.length) => {
  const empty = EditorState.create({
    extensions: [markdown({ extensions: GFM, codeLanguages: LANGUAGES }), wholeLines],
  })
  const state = empty.update({
    changes: { from: 0, insert: doc },
    selection: { anchor: Math.min(caret, doc.length) },
  }).state
  ensureSyntaxTree(state, doc.length, 5000)
  return state
}

const describeMarks = (state: EditorState, source: 'field' | 'full') => {
  const activeSet = source === 'field' ? state.field(wholeLines).marks : blockMarks(state)
  const found: unknown[] = []
  activeSet.between(0, state.doc.length, (from, to, mark) => {
    const widget = mark.spec.widget as { table?: unknown; isWritable?: boolean } | undefined
    found.push({
      from,
      to,
      kind: widget?.constructor.name,
      table: widget?.table,
      isWritable: widget?.isWritable,
    })
  })
  return found
}

const expectRebuiltEqual = (state: EditorState) =>
  expect(describeMarks(state, 'field')).toEqual(describeMarks(state, 'full'))

const apply = (state: EditorState, changes?: ChangeSpec, caret?: number) =>
  state.update({
    ...(changes === undefined ? {} : { changes }),
    ...(caret === undefined ? {} : { selection: { anchor: caret } }),
  }).state

const getRules = (state: EditorState) => {
  const rules: unknown[] = []
  state.field(wholeLines).marks.between(0, state.doc.length, (_from, _to, mark) => {
    if (mark.spec.widget instanceof Rule) rules.push(mark.spec.widget)
  })
  return rules
}

const isRedrawn = (before: readonly unknown[], after: readonly unknown[]) =>
  before.length > 0 && !before.some((rule) => after.includes(rule))

// The parser reparses an edit within a time budget read from Date.now, and a tree left partial
// changes the tables found. A clock that does not move gives it all the time it needs.
beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('the marks of whole lines', () => {
  const doc = `first\n\n${TABLE}\n\n---\n\nlast`

  it('are not drawn again for typing in a paragraph away from every table', () => {
    const state = create(doc, 3)
    const before = getRules(state)
    const next = apply(state, { from: 3, insert: 'xyz' })
    expect(before.length).toBeGreaterThan(0)
    expect(getRules(next)).toHaveLength(before.length)
    expect(getRules(next).every((rule, index) => rule === before[index])).toBe(true)
    expectRebuiltEqual(next)
  })

  it('are drawn again when the parser leaves the tables after an edit unparsed', () => {
    const state = create(doc, 3)
    const before = getRules(state)
    let now = 0
    vi.spyOn(Date, 'now').mockImplementation(() => (now += 1000))
    const next = apply(state, { from: 3, insert: 'xyz' })
    expect(isRedrawn(before, getRules(next))).toBe(true)
  })

  it('keep the position of a table that typing above it has moved', () => {
    const next = apply(create(doc, 3), { from: 3, insert: 'xyz' })
    const [grid] = describeMarks(next, 'field') as {
      table: { from: number; head: { from: number }[] }
    }[]
    expect(grid?.table.from).toBe(next.doc.toString().indexOf('| a |'))
    expect(grid?.table.head[0]?.from).toBe(next.doc.toString().indexOf('a |'))
  })

  it('are not drawn again for a caret that moves between paragraphs', () => {
    const state = create(doc, 1)
    const before = state.field(wholeLines)
    const next = apply(state, undefined, doc.length)
    expect(next.field(wholeLines)).toBe(before)
  })

  it.each([
    [
      'typing inside a table',
      (s: EditorState) => apply(s, { from: s.doc.toString().indexOf('1 |'), insert: 'x' }),
    ],
    [
      'a table added',
      (s: EditorState) => apply(s, { from: s.doc.length, insert: `\n\n${OTHER}\n\nend` }),
    ],
    [
      'a table deleted',
      (s: EditorState) => {
        const pos = s.doc.toString().indexOf(TABLE)
        return apply(s, { from: pos, to: pos + TABLE.length })
      },
    ],
    [
      'a caret entering a table',
      (s: EditorState) => apply(s, undefined, s.doc.toString().indexOf('| 1 |') + 2),
    ],
    ['a rule added', (s: EditorState) => apply(s, { from: 5, insert: '\n\n***' })],
    [
      'a rule removed',
      (s: EditorState) => {
        const pos = s.doc.toString().indexOf('---\n')
        return apply(s, { from: pos, to: pos + 4 })
      },
    ],
    ['a fence opened above a table', (s: EditorState) => apply(s, { from: 0, insert: '```\n' })],
  ])('are drawn again for %s, as a whole rebuild draws them', (_name, edit) => {
    const state = create(doc, 3)
    const before = getRules(state)
    const next = edit(state)
    expect(isRedrawn(before, getRules(next))).toBe(true)
    expectRebuiltEqual(next)
  })

  it('are drawn again when a caret leaves a table', () => {
    const inside = create(doc, doc.indexOf('| 1 |') + 2)
    expectRebuiltEqual(inside)
    const before = getRules(inside)
    const next = apply(inside, undefined, 1)
    expect(isRedrawn(before, getRules(next))).toBe(true)
    expectRebuiltEqual(next)
    expect(describeMarks(next, 'field').length).toBeGreaterThan(
      describeMarks(inside, 'field').length,
    )
  })

  it('are drawn again when a change merges two tables', () => {
    const two = `${TABLE}\n\n${OTHER}\n\ntail`
    const state = create(two, two.length)
    const pos = two.indexOf(OTHER)
    const next = apply(state, { from: TABLE.length, to: pos, insert: '\n' })
    expectRebuiltEqual(next)
  })

  it('are drawn for tables standing in a list and in a quote', () => {
    const nested = `- item\n\n  ${TABLE.replaceAll('\n', '\n  ')}\n\n> ${TABLE.replaceAll('\n', '\n> ')}\n\nend`
    const state = create(nested, nested.length)
    expectRebuiltEqual(state)
    expectRebuiltEqual(apply(state, { from: nested.length - 1, insert: 'x' }))
    expectRebuiltEqual(apply(state, { from: 0, to: 2 }))
  })
})

describe('the marks of whole lines under edits', () => {
  const FRAGMENTS = [
    'word ',
    '\n',
    '\n\n',
    `${TABLE}\n`,
    '| x | y |\n|---|---|\n',
    '|',
    '---\n',
    '***\n',
    '```\n',
    '> ',
    '- ',
    '  ',
    '# ',
  ]

  /** A small deterministic generator, so a failure repeats. */
  const createRandom = (seed: number) => {
    let value = seed
    return (limit: number) => {
      value = (value + 0x6d2b79f5) | 0
      let t = Math.imul(value ^ (value >>> 15), 1 | value)
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
      return Math.floor((((t ^ (t >>> 14)) >>> 0) / 4294967296) * limit)
    }
  }

  const SEEDS = [1, 2, 3, 4, 5, 6, 7, 8]
  const STEPS = 120

  it.each(SEEDS)(
    'equal a whole rebuild after every one of a run of random edits (seed %i)',
    (seed) => {
      const random = createRandom(seed)
      let state = create(
        `intro\n\n${TABLE}\n\n---\n\nouter\n\n- ${TABLE.replaceAll('\n', '\n  ')}\n`,
        0,
      )
      for (let step = 0; step < STEPS; step++) {
        const length = state.doc.length
        const pos = random(length + 1)
        const kind = random(3)
        if (kind === 0) {
          state = apply(state, { from: pos, insert: FRAGMENTS[random(FRAGMENTS.length)] as string })
        } else if (kind === 1) {
          const to = Math.min(length, pos + random(12))
          state = apply(state, { from: pos, to })
        } else {
          state = apply(state, undefined, pos)
        }
        expectRebuiltEqual(state)
      }
    },
  )
})
