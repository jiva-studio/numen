import { describe, expect, it } from 'vitest'
import { getTargetFolder } from './getTargetFolder'

describe('the folder a note from a template is made in', () => {
  it('makes the note inside a folder target', () => {
    expect(getTargetFolder({ path: 'notes/People', isFolder: true })).toBe('notes/People')
    expect(getTargetFolder({ path: 'notes', isFolder: true })).toBe('notes')
  })

  it('makes the note in the folder of a note target', () => {
    expect(getTargetFolder({ path: 'notes/a.md', isFolder: false })).toBe('notes')
    expect(getTargetFolder({ path: 'notes/People/Ada Lovelace.md', isFolder: false })).toBe(
      'notes/People',
    )
  })

  it('makes the note at the root for the root row', () => {
    expect(getTargetFolder({ path: '', isFolder: true })).toBe('')
    expect(getTargetFolder({ path: 'a.md', isFolder: false })).toBe('')
  })

  it('makes the note at the root when the target has no path', () => {
    expect(getTargetFolder({ path: '', isFolder: false })).toBe('')
  })
})
