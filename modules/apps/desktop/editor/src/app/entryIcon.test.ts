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
import { getEntryIcon } from './entryIcon'

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
    expect(getEntryIcon(null, false)).toBe(File)
    expect(getEntryIcon(undefined, true)).toBe(File)
  })

  it('is a closed folder for a folder that is shut', () => {
    expect(getEntryIcon(entry({ isFolder: true, kind: 'other' }), false)).toBe(Folder)
  })

  it('is an open folder for a folder that is open', () => {
    expect(getEntryIcon(entry({ isFolder: true, kind: 'other' }), true)).toBe(FolderOpen)
  })

  it('is a folder even where the folder is named for a note', () => {
    expect(getEntryIcon(entry({ isFolder: true, kind: 'note' }), false)).toBe(Folder)
  })

  it('is drawn by the kind of note for a note', () => {
    expect(getEntryIcon(entry({ type: 'note' }), false)).toBe(FileText)
    expect(getEntryIcon(entry({ type: 'deck' }), false)).toBe(Layers)
    expect(getEntryIcon(entry({ type: 'stencil' }), false)).toBe(LayoutTemplate)
    expect(getEntryIcon(entry({ type: 'preset' }), false)).toBe(Gauge)
  })

  it('is drawn by the source for a file that is not a note', () => {
    expect(getEntryIcon(entry({ kind: 'book' }), false)).toBe(BookOpen)
    expect(getEntryIcon(entry({ kind: 'recording' }), false)).toBe(AudioLines)
    expect(getEntryIcon(entry({ kind: 'url' }), false)).toBe(Globe)
  })

  it('is a plain file for a file the vault holds no source for', () => {
    expect(getEntryIcon(entry({ kind: 'other', type: 'deck' }), false)).toBe(File)
  })
})
