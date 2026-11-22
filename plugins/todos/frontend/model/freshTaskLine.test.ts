import { describe, expect, it } from 'vitest'
import { task } from '../testing/fakeApi'
import { freshTaskLine } from './freshTaskLine'

const at = (line: number, text = 'fix the tap') =>
  task({ path: 'a.md', line, proseLine: line - 3, text })

describe('the fresh task of a stored task', () => {
  it('follows the task to where the note holds it now', () => {
    expect(freshTaskLine([at(11)], at(7))).toEqual(at(11))
  })

  it('takes the nearest of several equal tasks', () => {
    expect(freshTaskLine([at(2), at(9), at(30)], at(8))).toEqual(at(9))
  })

  it('finds no task where none is the same', () => {
    expect(freshTaskLine([at(3, 'other')], at(7))).toBeUndefined()
    expect(freshTaskLine([], at(7))).toBeUndefined()
  })
})
