/**
 * How a command's message becomes the single line the window displays.
 */
import { describe, expect, it } from 'vitest'
import { formatNames, joinAll } from './voice'

describe('everything one command has to say', () => {
  it('is one line, each sentence ended by a full stop and a space', () => {
    expect(joinAll('One', 'Two', 'Three')).toBe('One. Two. Three')
  })

  it('leaves out what is empty', () => {
    expect(joinAll('', 'One', '', 'Two')).toBe('One. Two')
    expect(joinAll('', '')).toBe('')
  })
})

describe('the notes a command changed', () => {
  it('are named once each after what was done', () => {
    expect(formatNames('Links broken in', ['A.md', 'B.md', 'A.md'])).toBe(
      'Links broken in A.md, B.md',
    )
  })

  it('are nothing to say where there are none', () => {
    expect(formatNames('Links broken in', [])).toBe('')
  })
})
