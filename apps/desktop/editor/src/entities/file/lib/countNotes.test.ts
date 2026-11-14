import { describe, expect, it } from 'vitest'
import { countNotes } from './countNotes'
import type { Entry } from '../types'

const note = (path: string): Entry => ({
  path,
  name: path.split('/').at(-1) ?? path,
  isFolder: false,
  isNote: true,
  type: 'markdown',
})
const asset = (path: string): Entry => ({ ...note(path), isNote: false })
const folder = (path: string): Entry => ({ ...note(path), isFolder: true, isNote: false })

const tree = new Map<string, readonly Entry[]>([
  ['', [note('a.md'), asset('cover.png'), folder('notes')]],
  ['notes', [note('notes/b.md'), note('notes/c.md'), folder('notes/deep')]],
  ['notes/deep', [note('notes/deep/d.md')]],
])

describe('counting the notes of the vault', () => {
  it('counts the notes of every folder below the root and not the other files', async () => {
    const visited: string[] = []

    const count = await countNotes((path) => {
      visited.push(path)
      return Promise.resolve(tree.get(path) ?? [])
    })

    expect(count).toBe(4)
    expect([...visited].sort()).toStrictEqual(['', 'notes', 'notes/deep'])
  })

  it('counts nothing for an empty vault', async () => {
    expect(await countNotes(() => Promise.resolve([]))).toBe(0)
  })

  it('fails when a folder cannot be listed', async () => {
    await expect(countNotes(() => Promise.reject(new Error('unreadable')))).rejects.toThrow(
      'unreadable',
    )
  })
})
