import { describe, expect, it } from 'vitest'
import { findConflictNotes } from './conflict-notes.js'

const files = [
  { path: 'Shopping list.md', text: 'milk' },
  { path: 'Shopping list (conflict 2026-11-23).md', text: 'bread\n' },
  { path: 'Shopping list (conflict 2026-11-23 2).md', text: 'eggs' },
  { path: 'Food/Shopping list (conflict 2026-11-23).md', text: 'rice' },
  { path: 'Shopping lists (conflict 2026-11-23).md', text: 'tea' },
]

describe('findConflictNotes', () => {
  it('finds the conflict notes of a note, with the number of a taken name', () => {
    expect(findConflictNotes(files, 'Shopping list')).toStrictEqual(['bread', 'eggs'])
  })

  it('finds the conflict notes of a note in a folder and none of another note', () => {
    expect(findConflictNotes(files, 'Food/Shopping list')).toStrictEqual(['rice'])
    expect(findConflictNotes(files, 'Plan')).toStrictEqual([])
  })

  it('reads a title that has the signs of a pattern as plain text', () => {
    expect(findConflictNotes([{ path: 'A+B (conflict 2026-11-23).md', text: 'x' }], 'A+B')).toStrictEqual(['x'])
  })
})
