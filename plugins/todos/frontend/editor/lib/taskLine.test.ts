import { describe, expect, it } from 'vitest'
import { createFenceTracker, readTaskLine } from './taskLine'

const kinds = (line: string) =>
  readTaskLine(line)?.fields.map((one) => [one.kind, line.slice(one.from, one.to)])

describe('readTaskLine', () => {
  it('reads an open and a done task, whatever the list marker', () => {
    expect(readTaskLine('- [ ] write')?.isDone).toBe(false)
    expect(readTaskLine('* [x] write')?.isDone).toBe(true)
    expect(readTaskLine('1. [X] write')?.isDone).toBe(true)
  })

  it('starts the task at the list marker and its text after the checkbox', () => {
    const task = readTaskLine('  > - [ ] write')
    expect(task?.markerFrom).toBe(4)
    expect(task?.textFrom).toBe(10)
  })

  it('reads no task from a line that is not one', () => {
    for (const line of ['- plain', '-[ ] glued', '- [] empty', '- [ ]', '- [ ]x', 'text - [ ] a'])
      expect(readTaskLine(line)).toBeNull()
  })

  it('finds the fields of the line', () => {
    expect(
      kinds('- [ ] (A) call mum due:2026-11-20 +home @phone rec:1w pri:B done:2026-11-01'),
    ).toEqual([
      ['priority', '(A)'],
      ['due', 'due:2026-11-20'],
      ['project', '+home'],
      ['context', '@phone'],
      ['recurrence', 'rec:1w'],
      ['priority', 'pri:B'],
      ['done', 'done:2026-11-01'],
    ])
  })

  it('takes a priority in brackets only first, and leaves a lone prefix alone', () => {
    expect(kinds('- [ ] call (A) due: + @')).toEqual([])
  })

  it('leaves the fields written in a code span alone', () => {
    expect(kinds('- [ ] run `due:2026-11-20 @x` now')).toEqual([])
  })
})

const findFencedLines = (lines: string[]) => lines.map(createFenceTracker())

describe('createFenceTracker', () => {
  it('marks the lines of a fenced block and its fences', () => {
    const lines = ['a', '```ts', '- [ ] x', '```', 'b', '~~~', 'c', '~~~~', 'd']
    expect(findFencedLines(lines)).toEqual([
      false,
      true,
      true,
      true,
      false,
      true,
      true,
      true,
      false,
    ])
  })

  it('keeps a block open until a fence as long as its opening', () => {
    expect(findFencedLines(['````', '```', 'x', '````', 'y'])).toEqual([
      true,
      true,
      true,
      true,
      false,
    ])
  })
})
