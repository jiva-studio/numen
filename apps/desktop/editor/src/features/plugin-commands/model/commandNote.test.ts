/**
 * What a plugin command runs over, read from the target and the kind of its file.
 */
import { describe, expect, it, vi } from 'vitest'
import { createCommandNoteReader } from './usePluginCommands'

describe('createCommandNoteReader', () => {
  it('reads the type and the properties of the note of the target', async () => {
    const fileKinds = vi.fn(
      async () =>
        new Map([
          [
            'links/talk.md',
            { isNote: true, type: 'link' as const, properties: { media: 'a.mp4' } },
          ],
        ]),
    )

    const note = await createCommandNoteReader({ fileKinds })({
      path: 'links/talk.md',
      title: 'talk',
      isFolder: true,
      others: ['links/other.md'],
      tab: 'tab-1',
    })

    expect(fileKinds).toHaveBeenCalledWith(['links/talk.md'])
    expect(note).toStrictEqual({
      path: 'links/talk.md',
      title: 'talk',
      type: 'link',
      properties: { media: 'a.mp4' },
      isFolder: true,
      others: ['links/other.md'],
      tab: 'tab-1',
    })
  })

  it('reads nothing for a target with no path and defaults optional target fields', async () => {
    const fileKinds = vi.fn(async () => new Map())

    const note = await createCommandNoteReader({ fileKinds })({
      path: '',
      title: '',
    } as Parameters<ReturnType<typeof createCommandNoteReader>>[0])

    expect(fileKinds).not.toHaveBeenCalled()
    expect(note).toStrictEqual({
      path: '',
      title: '',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })
    expect(note).not.toHaveProperty('properties')
  })
})
