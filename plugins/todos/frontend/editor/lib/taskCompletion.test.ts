import { describe, expect, it } from 'vitest'
import { CompletionContext, type CompletionResult } from '@codemirror/autocomplete'
import { EditorState } from '@codemirror/state'
import { createTaskCompletion } from './taskCompletion'

const complete = createTaskCompletion({
  getToday: () => '2026-11-21',
  getFields: () => ({
    projects: [
      { name: 'home', count: 3 },
      { name: 'work', count: 2 },
    ],
    contexts: [{ name: 'phone', count: 1 }],
    tags: [
      { name: 'port', count: 2 },
      { name: 'Projects/Web', count: 1 },
    ],
  }),
})

function run(doc: string): CompletionResult | null {
  const pos = doc.indexOf('|')
  const state = EditorState.create({ doc: doc.replace('|', '') })
  return complete(new CompletionContext(state, pos, true)) as CompletionResult | null
}

const labels = (result: CompletionResult | null) => result?.options.map((one) => one.label)

describe('the task completion', () => {
  it('offers the projects after a plus and the contexts after an at sign', () => {
    expect(labels(run('- [ ] pay +|'))).toEqual(['home', 'work'])
    expect(labels(run('- [x] call @|'))).toEqual(['phone'])
  })

  it('offers the tags after a hash', () => {
    expect(labels(run('- [ ] pay #|'))).toEqual(['port', 'Projects/Web'])
    expect(labels(run('- [ ] pay #pro|'))).toEqual(['Projects/Web'])
  })

  it('filters by what is typed and replaces from after the sign', () => {
    const result = run('- [ ] pay +wo|')

    expect(labels(result)).toEqual(['work'])
    expect(result?.from).toBe('- [ ] pay +'.length)
  })

  it('offers the date shortcuts after due: and done: and applies the date', () => {
    const result = run('- [ ] pay due:to|')

    expect(labels(result)).toEqual(['today', 'tomorrow'])
    expect(result?.options.map((one) => one.apply)).toEqual(['2026-11-21', '2026-11-22'])
    expect(labels(run('- [x] pay done:|'))).toContain('+1w')
  })

  it('offers the letters after pri:', () => {
    expect(labels(run('- [ ] pay pri:|'))).toHaveLength(26)
    expect(labels(run('- [ ] pay pri:c|'))).toEqual(['C'])
  })

  it('answers nowhere but the text of a task line', () => {
    expect(run('plain +|')).toBeNull()
    expect(run('- plain +|')).toBeNull()
    expect(run('- [ ] +|')?.options).toHaveLength(2)
    expect(run('+|')).toBeNull()
    expect(run('- [ ] pay home|')).toBeNull()
    expect(run('- [ ] see [[+|')).toBeNull()
    expect(run('- [ ] see [[no|')).toBeNull()
  })

  it('answers nothing in a fenced code block', () => {
    expect(run('```\n- [ ] pay +|\n```')).toBeNull()
    expect(labels(run('```\ncode\n```\n- [ ] pay +|'))).toEqual(['home', 'work'])
  })

  it('answers nothing when no field matches', () => {
    expect(run('- [ ] pay +zzz|')).toBeNull()
  })
})
