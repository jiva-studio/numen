/**
 * What a plugin command runs over, read from the target and the kind of its file.
 */
import { describe, expect, it, vi } from 'vitest'
import { createCommandNoteReader } from './usePluginCommands'

const TARGET = { path: 'links/talk.md', title: 'talk', isFolder: false, others: [], tab: '' }

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

    const note = await createCommandNoteReader({ fileKinds })(TARGET)

    expect(note).toMatchObject({
      path: 'links/talk.md',
      type: 'link',
      properties: { media: 'a.mp4' },
    })
  })

  it('reads nothing for a target with no path', async () => {
    const fileKinds = vi.fn(async () => new Map())

    const note = await createCommandNoteReader({ fileKinds })({ ...TARGET, path: '' })

    expect(fileKinds).not.toHaveBeenCalled()
    expect(note).toMatchObject({ path: '', type: 'note' })
    expect(note).not.toHaveProperty('properties')
  })
})
