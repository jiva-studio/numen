import { describe, expect, it } from 'vitest'
import { task } from '../testing/fakeApi'
import { freshTaskLine } from './freshTaskLine'

const at = (line: number, text = 'fix the tap') => task({ path: 'a.md', line, text })

describe('the fresh line of a task', () => {
  it('follows the task to where the note holds it now', () => {
    expect(freshTaskLine([at(11)], at(7))).toBe(11)
  })

  it('takes the nearest of several equal tasks', () => {
    expect(freshTaskLine([at(2), at(9), at(30)], at(8))).toBe(9)
  })

  it('keeps the stored line where no task is the same', () => {
    expect(freshTaskLine([at(3, 'other')], at(7))).toBe(7)
    expect(freshTaskLine([], at(7))).toBe(7)
  })
})
