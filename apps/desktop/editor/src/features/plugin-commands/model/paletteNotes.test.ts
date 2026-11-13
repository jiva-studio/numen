/** The notes the palette reads for the conditions of the plugins' commands. Every note is invented. */
import { describe, expect, it } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import type { CommandTarget } from '@/shared/model/command'
import { createPaletteNotes } from './paletteNotes'

const target = (path: string): CommandTarget => ({
  tab: 'tab-1',
  kind: 'note',
  path,
  title: '',
  file: path,
  vault: { id: 'reef', name: 'Reef' },
  isReady: true,
})

const createNotes = () => {
  const reads: string[] = []
  let media = ''
  const notes = createPaletteNotes(async (one) => {
    reads.push(one.path)
    return { type: 'link', properties: { media } }
  })
  return { notes, reads, setMedia: (value: string) => (media = value) }
}

describe('the notes the palette reads', () => {
  it('is undefined while the note is read, then the type and properties of the note', async () => {
    const { notes } = createNotes()

    expect(notes.getNote(target('links/Mill.md'))).toBeUndefined()
    await flushPromises()

    expect(notes.getNote(target('links/Mill.md'))).toStrictEqual({
      type: 'link',
      properties: { media: '' },
    })
  })

  it('reads a note once until reset, and reads it again after', async () => {
    const { notes, reads, setMedia } = createNotes()

    notes.getNote(target('links/Mill.md'))
    notes.getNote(target('links/Mill.md'))
    await flushPromises()
    setMedia('media/mill.mp4')
    notes.reset()
    notes.getNote(target('links/Mill.md'))
    await flushPromises()

    expect(reads).toStrictEqual(['links/Mill.md', 'links/Mill.md'])
    expect(notes.getNote(target('links/Mill.md'))?.properties).toStrictEqual({
      media: 'media/mill.mp4',
    })
  })

  it('drops a read that ends after a reset', async () => {
    const { notes } = createNotes()

    notes.getNote(target('links/Mill.md'))
    notes.reset()
    await flushPromises()

    expect(notes.getNote(target('links/Mill.md'))).toBeUndefined()
  })
})
