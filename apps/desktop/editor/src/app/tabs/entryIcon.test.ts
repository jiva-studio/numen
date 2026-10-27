/**
 * The icon one entry of the files tree is drawn as.
 */
import { describe, expect, it } from 'vitest'
import { AudioLines, File, FileText, Folder, FolderOpen, Layers } from '@lucide/vue'
import type { Entry } from '@/entities/file'
import { registerNoteIcon } from '@/entities/note'
import { getEntryIcon } from './entryIcon'

const entry = (overrides: Partial<Entry> = {}): Entry => ({
  path: 'a',
  name: 'a',
  isFolder: false,
  isNote: true,
  type: 'note',
  ...overrides,
})

describe('the icon of an entry', () => {
  it('is a plain file where there is no entry', () => {
    expect(getEntryIcon(null, false)).toBe(File)
    expect(getEntryIcon(undefined, true)).toBe(File)
  })

  it('is a closed folder for a folder that is shut', () => {
    expect(getEntryIcon(entry({ isFolder: true, isNote: false }), false)).toBe(Folder)
  })

  it('is an open folder for a folder that is open', () => {
    expect(getEntryIcon(entry({ isFolder: true, isNote: false }), true)).toBe(FolderOpen)
  })

  it('is a folder even where the folder is named for a note', () => {
    expect(getEntryIcon(entry({ isFolder: true, isNote: true }), false)).toBe(Folder)
  })

  it('is drawn by the type of note for a note, as a plugin registers its icon', () => {
    expect(getEntryIcon(entry({ type: 'note' }), false)).toBe(FileText)
    expect(getEntryIcon(entry({ type: 'sample' }), false)).toBe(FileText)

    const unregister = registerNoteIcon('sample', Layers)
    expect(getEntryIcon(entry({ type: 'sample' }), false)).toBe(Layers)
    unregister()
  })

  it('is the icon of the reader that opens a file that is no note', () => {
    const findFileIcon = (path: string) => (path === 'talk.bin' ? AudioLines : undefined)

    expect(getEntryIcon(entry({ path: 'talk.bin', isNote: false }), false, findFileIcon)).toBe(
      AudioLines,
    )
  })

  it('is a plain file for a file no reader opens', () => {
    expect(getEntryIcon(entry({ isNote: false, type: 'sample' }), false, () => undefined)).toBe(File)
  })
})
