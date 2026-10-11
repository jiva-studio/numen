/**
 * The icon one entry of the files tree is drawn as.
 */
import { describe, expect, it } from 'vitest'
import {
  AudioLines,
  BookOpen,
  File,
  FileText,
  Folder,
  FolderOpen,
  Gauge,
  Globe,
  Layers,
  LayoutTemplate,
} from '@lucide/vue'
import type { Entry } from '@/entities/file'
import { iconOfEntry } from './icons'

const entry = (over: Partial<Entry> = {}): Entry => ({
  path: 'a',
  name: 'a',
  isFolder: false,
  kind: 'note',
  type: 'note',
  ...over,
})

describe('the icon of an entry', () => {
  it('is a plain file where there is no entry', () => {
    expect(iconOfEntry(null, false)).toBe(File)
    expect(iconOfEntry(undefined, true)).toBe(File)
  })

  it('is a closed folder for a folder that is shut', () => {
    expect(iconOfEntry(entry({ isFolder: true, kind: 'other' }), false)).toBe(Folder)
  })

  it('is an open folder for a folder that is open', () => {
    expect(iconOfEntry(entry({ isFolder: true, kind: 'other' }), true)).toBe(FolderOpen)
  })

  it('is a folder even where the folder is named for a note', () => {
    expect(iconOfEntry(entry({ isFolder: true, kind: 'note' }), false)).toBe(Folder)
  })

  it('is drawn by the kind of note for a note', () => {
    expect(iconOfEntry(entry({ type: 'note' }), false)).toBe(FileText)
    expect(iconOfEntry(entry({ type: 'deck' }), false)).toBe(Layers)
    expect(iconOfEntry(entry({ type: 'stencil' }), false)).toBe(LayoutTemplate)
    expect(iconOfEntry(entry({ type: 'preset' }), false)).toBe(Gauge)
  })

  it('is drawn by the source for a file that is not a note', () => {
    expect(iconOfEntry(entry({ kind: 'book' }), false)).toBe(BookOpen)
    expect(iconOfEntry(entry({ kind: 'recording' }), false)).toBe(AudioLines)
    expect(iconOfEntry(entry({ kind: 'url' }), false)).toBe(Globe)
  })

  it('is a plain file for a file the vault holds no source for', () => {
    expect(iconOfEntry(entry({ kind: 'other', type: 'deck' }), false)).toBe(File)
  })
})
