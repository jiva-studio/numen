import { describe, expect, it } from 'vitest'
import { createTask } from '../testing/fakeApi'
import { createTaskLine } from './freshTaskLine'

const at = (line: number, text = 'fix the tap') =>
  createTask({ path: 'a.md', line, proseLine: line - 3, text })

describe('the fresh task of a stored task', () => {
  it('follows the task to where the note holds it now', () => {
    expect(createTaskLine([at(11)], at(7))).toEqual(at(11))
  })

  it('takes the nearest of several equal tasks', () => {
    expect(createTaskLine([at(2), at(9), at(30)], at(8))).toEqual(at(9))
  })

  it('finds no task where none is the same', () => {
    expect(createTaskLine([at(3, 'other')], at(7))).toBeUndefined()
    expect(createTaskLine([], at(7))).toBeUndefined()
  })
})
