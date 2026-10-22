import { describe, expect, it, vi, beforeEach } from 'vitest'
import { markdown } from '@codemirror/lang-markdown'
import { ensureSyntaxTree } from '@codemirror/language'
import { EditorState, type ChangeSpec } from '@codemirror/state'
import { GFM } from '@lezer/markdown'
import { LANGUAGES } from '../config/languages'
import { wholeLines } from './live'
import { blockMarks } from './marks'

vi.mock('./marks', async (importOriginal) => {
  const original = await importOriginal<typeof import('./marks')>()
  return { ...original, blockMarks: vi.fn(original.blockMarks) }
})

const TABLE = '| a | b |\n|---|---|\n| 1 | 2 |'
const OTHER = '| c | d |\n|---|---|\n| 3 | 4 |'

/** A document written into a state in one transaction, as a file opened is. */
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

/** The decorations as plain data: where they stand and what they stand for. */
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

const calls = () => vi.mocked(blockMarks).mock.calls.length

beforeEach(() => {
  vi.mocked(blockMarks).mockClear()
})

describe('the marks of whole lines', () => {
  const doc = `first\n\n${TABLE}\n\n---\n\nlast`

  it('are not drawn again for typing in a paragraph away from every table', () => {
    const state = create(doc, 3)
    const before = calls()
    const next = apply(state, { from: 3, insert: 'xyz' })
    expect(calls()).toBe(before)
    expectRebuiltEqual(next)
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
    const before = calls()
    const next = edit(state)
    expect(calls()).toBeGreaterThan(before)
    expectRebuiltEqual(next)
  })

  it('are drawn again when a caret leaves a table', () => {
    const inside = create(doc, doc.indexOf('| 1 |') + 2)
    expectRebuiltEqual(inside)
    const before = calls()
    const next = apply(inside, undefined, 1)
    expect(calls()).toBeGreaterThan(before)
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
