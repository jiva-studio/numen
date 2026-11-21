/** The notes the palette reads for the conditions of the plugins' commands. Every note is invented. */
import { describe, expect, it } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import type { CommandTarget } from '@/shared/model/command'
import { createPaletteNotes } from './paletteNotes'

const target = (path: string, override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab-1',
  kind: 'note',
  path,
  title: 'Mill',
  file: path,
  vault: { id: 'reef', name: 'Reef' },
  isReady: true,
  ...override,
})

const createNotes = () => {
  const reads: string[] = []
  const targets: Parameters<Parameters<typeof createPaletteNotes>[0]['read']>[0][] = []
  const errors: unknown[] = []
  let media = ''
  let failure: Error | undefined
  const notes = createPaletteNotes({
    read: async (one) => {
      reads.push(one.path)
      targets.push(one)
      if (failure) throw failure
      return { type: 'link', properties: { media } }
    },
    reportError: (error) => errors.push(error),
  })
  return {
    notes,
    reads,
    targets,
    errors,
    setMedia: (value: string) => (media = value),
    setFailure: (value: Error | undefined) => (failure = value),
  }
}

describe('the notes the palette reads', () => {
  it('is undefined while the note is read, then the type and properties of the note', async () => {
    const { notes, targets } = createNotes()

    expect(notes.getNote(target('links/Mill.md', { path: '' }))).toBeUndefined()
    expect(notes.getNote(target('links/Other.md', { others: ['links/Extra.md'] }))).toBeUndefined()
    await flushPromises()

    expect(notes.getNote(target('links/Mill.md', { path: '' }))).toStrictEqual({
      type: 'link',
      properties: { media: '' },
    })
    expect(targets).toStrictEqual([
      { path: 'links/Mill.md', title: 'Mill', isFolder: false, others: [] },
      { path: 'links/Other.md', title: 'Mill', isFolder: false, others: ['links/Extra.md'] },
    ])
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

  it('reports a failed read and reads the note again on the next ask', async () => {
    const { notes, reads, errors, setFailure } = createNotes()
    const failure = new Error('the vault is closed')
    setFailure(failure)

    notes.getNote(target('links/Mill.md'))
    await flushPromises()
    setFailure(undefined)
    notes.getNote(target('links/Mill.md'))
    await flushPromises()

    expect(errors).toStrictEqual([failure])
    expect(reads).toStrictEqual(['links/Mill.md', 'links/Mill.md'])
    expect(notes.getNote(target('links/Mill.md'))?.type).toBe('link')
  })

  it('drops a read that ends after a reset', async () => {
    const { notes, errors, setFailure } = createNotes()

    notes.getNote(target('links/Mill.md'))
    notes.reset()
    await flushPromises()

    expect(notes.getNote(target('links/Mill.md'))).toBeUndefined()

    setFailure(new Error('closed'))
    notes.getNote(target('links/Mill.md'))
    notes.reset()
    await flushPromises()

    expect(errors).toStrictEqual([])
  })
})
