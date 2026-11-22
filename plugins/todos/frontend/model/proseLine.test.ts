import { describe, expect, it } from 'vitest'
import { proseLine } from './proseLine'

const PROSE = '# Heading\n\nprose\n- [ ] the task\nafter\n- [ ] other'

describe('the line of a task in the prose of its note', () => {
  it('drops the lines of the frontmatter the file has above the prose', () => {
    const target = { line: 7, raw: '- [ ] the task' }
    expect(proseLine(PROSE, [target], target)).toBe(4)
  })

  it('is the line of the file where the note has no frontmatter', () => {
    const target = { line: 4, raw: '- [ ] the task' }
    expect(proseLine(PROSE, [target], target)).toBe(4)
  })

  it('tells a task from its twin by the other tasks of the note', () => {
    const body = '- [ ] same\n- [ ] same\nmid\n- [ ] other'
    const tasks = [
      { line: 4, raw: '- [ ] same' },
      { line: 5, raw: '- [ ] same' },
      { line: 7, raw: '- [ ] other' },
    ]
    expect(proseLine(body, tasks, tasks[1] as (typeof tasks)[number])).toBe(2)
  })

  it('reads a line that ends in a carriage return as the task has it', () => {
    const target = { line: 5, raw: '- [ ] the task' }
    expect(proseLine('a\r\n- [ ] the task\r\n', [target], target)).toBe(2)
  })

  it('keeps the stored line where the prose has no such task', () => {
    const target = { line: 7, raw: '- [ ] gone' }
    expect(proseLine(PROSE, [target], target)).toBe(7)
  })
})
